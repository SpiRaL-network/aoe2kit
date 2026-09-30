package geotrace

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const worldCoverBaseURL = "https://esa-worldcover.s3.eu-central-1.amazonaws.com/v200/2021/map"
const glcFCS30D2022URL = "https://s3.openlandmap.org/arco/lc_glc.fcs30d_c_30m_s_20220101_20221231_go_epsg.4326_v20231026.tif"

var errWorldCoverTileMissing = errors.New("worldcover tile missing")

func sampleLandcover(opts Options, points []Cell, landmask []bool) ([]LandcoverSample, string, error) {
	provider := strings.ToLower(strings.TrimSpace(opts.LandcoverProvider))
	if provider == "" && (opts.LandcoverPath != "" || opts.LandcoverDir != "" || opts.LandcoverCacheDir != "") {
		provider = "worldcover"
	}
	if provider == "" {
		return nil, "", nil
	}
	if provider != "worldcover" && provider != "glc-fcs30d" && provider != "glc_fcs30d" {
		return nil, "", fmt.Errorf("unsupported landcover provider %q", opts.LandcoverProvider)
	}
	if provider == "glc_fcs30d" {
		provider = "glc-fcs30d"
	}
	sampler := &worldCoverSampler{
		provider: provider,
		client:   opts.Client,
		cacheDir: opts.LandcoverCacheDir,
		filePath: opts.LandcoverPath,
		fileDir:  opts.LandcoverDir,
	}
	out := make([]LandcoverSample, len(points))
	for i, p := range points {
		if len(landmask) == len(points) && !landmask[i] {
			continue
		}
		sample, err := sampler.sample(p.Lat, p.Lon)
		if err != nil {
			return nil, "", err
		}
		out[i] = sample
	}
	return out, provider, nil
}

type worldCoverSampler struct {
	provider string
	client   *http.Client
	cacheDir string
	filePath string
	fileDir  string
	tiles    map[string]*worldCoverTile
	missing  map[string]bool
}

func (s *worldCoverSampler) sample(lat, lon float64) (LandcoverSample, error) {
	if s.provider == "glc-fcs30d" {
		return s.sampleGLC(lat, lon)
	}
	name := worldCoverTileName(lat, lon)
	if name == "" {
		return LandcoverSample{}, nil
	}
	if s.tiles == nil {
		s.tiles = map[string]*worldCoverTile{}
	}
	if s.missing == nil {
		s.missing = map[string]bool{}
	}
	if s.missing[name] {
		return LandcoverSample{}, nil
	}
	tile := s.tiles[name]
	if tile == nil {
		loaded, err := s.loadTile(name)
		if err != nil {
			if errors.Is(err, errWorldCoverTileMissing) {
				s.missing[name] = true
				return LandcoverSample{}, nil
			}
			return LandcoverSample{}, err
		}
		tile = loaded
		s.tiles[name] = tile
	}
	return tile.sample(lat, lon)
}

func (s *worldCoverSampler) sampleGLC(lat, lon float64) (LandcoverSample, error) {
	const name = "GLC_FCS30D_2022_global_cog.tif"
	if s.tiles == nil {
		s.tiles = map[string]*worldCoverTile{}
	}
	tile := s.tiles[name]
	if tile == nil {
		loaded, err := s.loadGLCTile(name)
		if err != nil {
			return LandcoverSample{}, err
		}
		tile = loaded
		s.tiles[name] = tile
	}
	return tile.sample(lat, lon)
}

func (s *worldCoverSampler) loadTile(name string) (*worldCoverTile, error) {
	src, err := s.sourceForTile(name)
	if err != nil {
		return nil, err
	}
	header, err := src.readAt(0, 128<<10)
	if err != nil {
		src.close()
		return nil, err
	}
	tile, err := parseWorldCoverTile(name, src, header)
	if err != nil {
		src.close()
		return nil, err
	}
	return tile, nil
}

func (s *worldCoverSampler) loadGLCTile(name string) (*worldCoverTile, error) {
	src, err := s.sourceForGLC(name)
	if err != nil {
		return nil, err
	}
	header, err := src.readAt(0, 256<<10)
	if err != nil {
		src.close()
		return nil, err
	}
	tile, err := parseLandcoverTile(name, src, header, false)
	if err != nil {
		src.close()
		return nil, err
	}
	tile.classKey = glcCoarseClassKey
	tile.glcClassKey = glcFCS30DClassKey
	return tile, nil
}

func (s *worldCoverSampler) sourceForTile(name string) (rangeSource, error) {
	if s.filePath != "" {
		return newFileRangeSource(s.filePath)
	}
	if s.fileDir != "" {
		return newFileRangeSource(filepath.Join(s.fileDir, name))
	}
	if s.cacheDir == "" {
		return nil, fmt.Errorf("worldcover requires --landcover-cache, --landcover-dir, or --landcover-file")
	}
	url := worldCoverBaseURL + "/" + name
	return &httpRangeSource{
		url:      url,
		client:   s.httpClient(),
		cacheDir: filepath.Join(s.cacheDir, "ranges", strings.TrimSuffix(name, ".tif")),
	}, nil
}

func (s *worldCoverSampler) sourceForGLC(name string) (rangeSource, error) {
	if s.filePath != "" {
		return newFileRangeSource(s.filePath)
	}
	if s.fileDir != "" {
		return newFileRangeSource(filepath.Join(s.fileDir, name))
	}
	if s.cacheDir == "" {
		return nil, fmt.Errorf("glc-fcs30d requires --landcover-cache, --landcover-dir, or --landcover-file")
	}
	return &httpRangeSource{
		url:      glcFCS30D2022URL,
		client:   s.httpClient(),
		cacheDir: filepath.Join(s.cacheDir, "ranges", "glc_fcs30d_2022"),
	}, nil
}

func (s *worldCoverSampler) httpClient() *http.Client {
	if s.client != nil {
		return s.client
	}
	return &http.Client{Timeout: 90 * time.Second}
}

type worldCoverTile struct {
	name        string
	src         rangeSource
	image       tiffImage
	tileCache   map[int][]byte
	tileOrder   []int
	classKey    func(int) string
	glcClassKey func(int) string
}

const landcoverRawTileCacheLimit = 128

func (t *worldCoverTile) sample(lat, lon float64) (LandcoverSample, error) {
	if lon < t.image.minLon || lon >= t.image.maxLon || lat <= t.image.minLat || lat > t.image.maxLat {
		return LandcoverSample{}, nil
	}
	px := int(math.Floor((lon - t.image.minLon) / t.image.pixelWidth))
	py := int(math.Floor((t.image.maxLat - lat) / t.image.pixelHeight))
	if px < 0 || py < 0 || px >= t.image.width || py >= t.image.height {
		return LandcoverSample{}, nil
	}
	counts := map[int]int{}
	total := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			xx := px + dx
			yy := py + dy
			if xx < 0 || yy < 0 || xx >= t.image.width || yy >= t.image.height {
				continue
			}
			class, err := t.pixel(xx, yy)
			if err != nil {
				return LandcoverSample{}, err
			}
			if class == 0 {
				continue
			}
			counts[class]++
			total++
		}
	}
	keyFn := t.classKey
	if keyFn == nil {
		keyFn = worldCoverClassKey
	}
	hist := sortedLandcoverStats(counts, total, keyFn)
	if len(hist) == 0 {
		return LandcoverSample{}, nil
	}
	sample := LandcoverSample{
		Valid:     true,
		Class:     hist[0].Class,
		Key:       hist[0].Key,
		Fraction:  hist[0].Weight,
		Histogram: hist,
	}
	if t.glcClassKey != nil {
		sample.GLCClass = hist[0].Class
		sample.GLCKey = t.glcClassKey(hist[0].Class)
	}
	return sample, nil
}

func (t *worldCoverTile) pixel(px, py int) (int, error) {
	tx := px / t.image.tileWidth
	ty := py / t.image.tileHeight
	tilesX := int(math.Ceil(float64(t.image.width) / float64(t.image.tileWidth)))
	tileIndex := ty*tilesX + tx
	if tileIndex < 0 || tileIndex >= len(t.image.tileOffsets) {
		return 0, fmt.Errorf("worldcover %s tile index %d out of range", t.name, tileIndex)
	}
	if t.tileCache == nil {
		t.tileCache = map[int][]byte{}
	}
	raw := t.tileCache[tileIndex]
	if raw == nil {
		offset := t.image.tileOffsets[tileIndex]
		count := t.image.tileByteCounts[tileIndex]
		compressed, err := t.src.readAt(offset, count)
		if err != nil {
			return 0, err
		}
		raw, err = inflateDeflateTile(compressed)
		if err != nil {
			return 0, fmt.Errorf("worldcover %s tile %d inflate: %w", t.name, tileIndex, err)
		}
		want := t.image.tileWidth * t.image.tileHeight
		if len(raw) < want {
			return 0, fmt.Errorf("worldcover %s tile %d decoded %d byte(s), want at least %d", t.name, tileIndex, len(raw), want)
		}
		t.cacheRawTile(tileIndex, raw)
	}
	localX := px % t.image.tileWidth
	localY := py % t.image.tileHeight
	return int(raw[localY*t.image.tileWidth+localX]), nil
}

func (t *worldCoverTile) cacheRawTile(tileIndex int, raw []byte) {
	if t.tileCache == nil {
		t.tileCache = map[int][]byte{}
	}
	if _, exists := t.tileCache[tileIndex]; !exists {
		t.tileOrder = append(t.tileOrder, tileIndex)
	}
	t.tileCache[tileIndex] = raw
	for len(t.tileOrder) > landcoverRawTileCacheLimit {
		evict := t.tileOrder[0]
		copy(t.tileOrder, t.tileOrder[1:])
		t.tileOrder = t.tileOrder[:len(t.tileOrder)-1]
		delete(t.tileCache, evict)
	}
}

type tiffImage struct {
	width          int
	height         int
	tileWidth      int
	tileHeight     int
	tileOffsets    []int64
	tileByteCounts []int64
	minLon         float64
	maxLat         float64
	maxLon         float64
	minLat         float64
	pixelWidth     float64
	pixelHeight    float64
}

func parseWorldCoverTile(name string, src rangeSource, header []byte) (*worldCoverTile, error) {
	return parseLandcoverTile(name, src, header, true)
}

func parseLandcoverTile(name string, src rangeSource, header []byte, allowOverview bool) (*worldCoverTile, error) {
	if len(header) < 16 || string(header[:2]) != "II" {
		return nil, fmt.Errorf("landcover %s is not a supported little-endian TIFF", name)
	}
	reader, err := newTIFFReader(header)
	if err != nil {
		return nil, fmt.Errorf("landcover %s: %w", name, err)
	}
	ifdOffset := reader.firstIFDOffset()
	var images []tiffImage
	var geo tiffGeo
	for ifdOffset != 0 {
		ifd, next, err := reader.readIFD(src, ifdOffset)
		if err != nil {
			return nil, err
		}
		img, ok, err := tiffImageFromIFD(src, header, ifd)
		if err != nil {
			return nil, err
		}
		if ok {
			if scale := tiffDoubles(src, header, ifd[33550]); len(scale) >= 2 {
				geo.pixelWidth = scale[0]
				geo.pixelHeight = math.Abs(scale[1])
			}
			if tie := tiffDoubles(src, header, ifd[33922]); len(tie) >= 6 {
				geo.minLon = tie[3]
				geo.maxLat = tie[4]
			}
			images = append(images, img)
		}
		ifdOffset = next
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("worldcover %s has no supported tiled image directories", name)
	}
	if geo.pixelWidth == 0 || geo.pixelHeight == 0 {
		return nil, fmt.Errorf("worldcover %s missing georeference tags", name)
	}
	best := images[0]
	if allowOverview {
		best = chooseWorldCoverOverview(images)
	}
	scaleFactor := float64(images[0].width) / float64(best.width)
	best.pixelWidth = geo.pixelWidth * scaleFactor
	best.pixelHeight = geo.pixelHeight * scaleFactor
	best.minLon = geo.minLon
	best.maxLat = geo.maxLat
	best.maxLon = best.minLon + float64(best.width)*best.pixelWidth
	best.minLat = best.maxLat - float64(best.height)*best.pixelHeight
	return &worldCoverTile{name: name, src: src, image: best, classKey: worldCoverClassKey}, nil
}

type tiffGeo struct {
	minLon      float64
	maxLat      float64
	pixelWidth  float64
	pixelHeight float64
}

type tiffTag struct {
	typ   uint16
	count uint64
	value uint64
}

type tiffReader struct {
	header []byte
	big    bool
}

func newTIFFReader(header []byte) (tiffReader, error) {
	magic := binary.LittleEndian.Uint16(header[2:4])
	switch magic {
	case 42:
		return tiffReader{header: header}, nil
	case 43:
		if len(header) < 16 {
			return tiffReader{}, fmt.Errorf("BigTIFF header too short")
		}
		if binary.LittleEndian.Uint16(header[4:6]) != 8 || binary.LittleEndian.Uint16(header[6:8]) != 0 {
			return tiffReader{}, fmt.Errorf("unsupported BigTIFF offset-size header")
		}
		return tiffReader{header: header, big: true}, nil
	default:
		return tiffReader{}, fmt.Errorf("unsupported TIFF magic %d", magic)
	}
}

func (r tiffReader) firstIFDOffset() int64 {
	if r.big {
		return int64(binary.LittleEndian.Uint64(r.header[8:16]))
	}
	return int64(binary.LittleEndian.Uint32(r.header[4:8]))
}

func (r tiffReader) readIFD(src rangeSource, offset int64) (map[uint16]tiffTag, int64, error) {
	if r.big {
		return r.readBigIFD(src, offset)
	}
	return r.readClassicIFD(src, offset)
}

func (r tiffReader) readClassicIFD(src rangeSource, offset int64) (map[uint16]tiffTag, int64, error) {
	need := int(offset) + 2
	if need > len(r.header) {
		return nil, 0, fmt.Errorf("TIFF IFD offset %d outside cached header", offset)
	}
	n := int(binary.LittleEndian.Uint16(r.header[offset : offset+2]))
	end := int(offset) + 2 + n*12 + 4
	data := r.header
	localOffset := offset
	if end > len(r.header) {
		var err error
		data, err = src.readAt(offset, int64(2+n*12+4))
		if err != nil {
			return nil, 0, err
		}
		localOffset = 0
	}
	tags := map[uint16]tiffTag{}
	pos := int(localOffset) + 2
	for i := 0; i < n; i++ {
		tag := binary.LittleEndian.Uint16(data[pos : pos+2])
		typ := binary.LittleEndian.Uint16(data[pos+2 : pos+4])
		count := uint64(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		value := uint64(binary.LittleEndian.Uint32(data[pos+8 : pos+12]))
		tags[tag] = tiffTag{typ: typ, count: count, value: value}
		pos += 12
	}
	next := int64(binary.LittleEndian.Uint32(data[pos : pos+4]))
	return tags, next, nil
}

func (r tiffReader) readBigIFD(src rangeSource, offset int64) (map[uint16]tiffTag, int64, error) {
	need := int(offset) + 8
	if need > len(r.header) {
		return nil, 0, fmt.Errorf("BigTIFF IFD offset %d outside cached header", offset)
	}
	n64 := binary.LittleEndian.Uint64(r.header[offset : offset+8])
	if n64 > 4096 {
		return nil, 0, fmt.Errorf("BigTIFF IFD entry count %d is too large", n64)
	}
	n := int(n64)
	end := int(offset) + 8 + n*20 + 8
	data := r.header
	localOffset := offset
	if end > len(r.header) {
		var err error
		data, err = src.readAt(offset, int64(8+n*20+8))
		if err != nil {
			return nil, 0, err
		}
		localOffset = 0
	}
	tags := map[uint16]tiffTag{}
	pos := int(localOffset) + 8
	for i := 0; i < n; i++ {
		tag := binary.LittleEndian.Uint16(data[pos : pos+2])
		typ := binary.LittleEndian.Uint16(data[pos+2 : pos+4])
		count := binary.LittleEndian.Uint64(data[pos+4 : pos+12])
		value := binary.LittleEndian.Uint64(data[pos+12 : pos+20])
		tags[tag] = tiffTag{typ: typ, count: count, value: value}
		pos += 20
	}
	next := int64(binary.LittleEndian.Uint64(data[pos : pos+8]))
	return tags, next, nil
}

func tiffImageFromIFD(src rangeSource, header []byte, tags map[uint16]tiffTag) (tiffImage, bool, error) {
	width := tiffInt(tags[256])
	height := tiffInt(tags[257])
	compression := tiffInt(tags[259])
	bits := tiffInt(tags[258])
	samples := tiffInt(tags[277])
	tileWidth := tiffInt(tags[322])
	tileHeight := tiffInt(tags[323])
	if width == 0 || height == 0 || tileWidth == 0 || tileHeight == 0 {
		return tiffImage{}, false, nil
	}
	if compression != 8 || bits != 8 || samples != 1 {
		return tiffImage{}, false, fmt.Errorf("unsupported landcover TIFF shape width=%d height=%d compression=%d bits=%d samples=%d", width, height, compression, bits, samples)
	}
	offsets, err := tiffIntArray(src, header, tags[324])
	if err != nil {
		return tiffImage{}, false, err
	}
	counts, err := tiffIntArray(src, header, tags[325])
	if err != nil {
		return tiffImage{}, false, err
	}
	if len(offsets) != len(counts) {
		return tiffImage{}, false, fmt.Errorf("tile offset/count mismatch %d/%d", len(offsets), len(counts))
	}
	return tiffImage{
		width:          width,
		height:         height,
		tileWidth:      tileWidth,
		tileHeight:     tileHeight,
		tileOffsets:    offsets,
		tileByteCounts: counts,
	}, true, nil
}

func chooseWorldCoverOverview(images []tiffImage) tiffImage {
	best := images[len(images)-1]
	for _, img := range images {
		if img.width <= 1200 {
			best = img
			break
		}
	}
	return best
}

func tiffInt(tag tiffTag) int {
	if tag.count == 0 {
		return 0
	}
	switch tag.typ {
	case 3:
		return int(tag.value & 0xffff)
	case 4:
		return int(tag.value)
	case 16:
		return int(tag.value)
	default:
		return 0
	}
}

func tiffIntArray(src rangeSource, header []byte, tag tiffTag) ([]int64, error) {
	if tag.count == 0 {
		return nil, nil
	}
	size := int64(4 * tag.count)
	if tag.typ == 3 {
		size = int64(2 * tag.count)
	} else if tag.typ == 16 {
		size = int64(8 * tag.count)
	}
	var data []byte
	inlineSize := int64(4)
	if tag.value > math.MaxUint32 || tag.typ == 16 {
		inlineSize = 8
	}
	if size <= inlineSize {
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, tag.value)
		data = buf[:size]
	} else if uint64(len(header)) >= tag.value+uint64(size) {
		data = header[tag.value : int64(tag.value)+size]
	} else {
		var err error
		data, err = src.readAt(int64(tag.value), size)
		if err != nil {
			return nil, err
		}
	}
	out := make([]int64, int(tag.count))
	for i := range out {
		if tag.typ == 3 {
			out[i] = int64(binary.LittleEndian.Uint16(data[i*2 : i*2+2]))
		} else if tag.typ == 4 {
			out[i] = int64(binary.LittleEndian.Uint32(data[i*4 : i*4+4]))
		} else {
			out[i] = int64(binary.LittleEndian.Uint64(data[i*8 : i*8+8]))
		}
	}
	return out, nil
}

func tiffDoubles(src rangeSource, header []byte, tag tiffTag) []float64 {
	if tag.typ != 12 || tag.count == 0 {
		return nil
	}
	size := int64(8 * tag.count)
	var data []byte
	inlineSize := int64(4)
	if tag.value > math.MaxUint32 {
		inlineSize = 8
	}
	if size <= inlineSize {
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, tag.value)
		data = buf[:size]
	} else if uint64(len(header)) >= tag.value+uint64(size) {
		data = header[tag.value : int64(tag.value)+size]
	} else {
		fetched, err := src.readAt(int64(tag.value), size)
		if err != nil {
			return nil
		}
		data = fetched
	}
	out := make([]float64, int(tag.count))
	for i := range out {
		out[i] = math.Float64frombits(binary.LittleEndian.Uint64(data[i*8 : i*8+8]))
	}
	return out
}

func inflateDeflateTile(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

type rangeSource interface {
	readAt(offset int64, length int64) ([]byte, error)
	close() error
}

type fileRangeSource struct {
	path string
	file *os.File
}

func newFileRangeSource(path string) (*fileRangeSource, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return &fileRangeSource{path: path, file: f}, nil
}

func (s *fileRangeSource) readAt(offset int64, length int64) ([]byte, error) {
	buf := make([]byte, length)
	n, err := s.file.ReadAt(buf, offset)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return buf[:n], nil
}

func (s *fileRangeSource) close() error {
	return s.file.Close()
}

type httpRangeSource struct {
	url      string
	client   *http.Client
	cacheDir string
}

func (s *httpRangeSource) readAt(offset int64, length int64) ([]byte, error) {
	if err := os.MkdirAll(s.cacheDir, 0o755); err != nil {
		return nil, err
	}
	cachePath := filepath.Join(s.cacheDir, fmt.Sprintf("%012d_%012d.bin", offset, length))
	if data, err := os.ReadFile(cachePath); err == nil && int64(len(data)) == length {
		return data, nil
	}
	req, err := http.NewRequest("GET", s.url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: %s", errWorldCoverTileMissing, s.url)
	}
	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("WorldCover range fetch %s status %d", s.url, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, length+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) < length {
		return nil, fmt.Errorf("WorldCover range fetch %s returned %d byte(s), want %d", s.url, len(data), length)
	}
	data = data[:length]
	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		return nil, err
	}
	return data, nil
}

func (s *httpRangeSource) close() error {
	return nil
}

func worldCoverTileName(lat, lon float64) string {
	if lat < -60 || lat > 84 || lon < -180 || lon > 180 {
		return ""
	}
	lat0 := math.Floor(lat/3) * 3
	lon0 := math.Floor(lon/3) * 3
	ns := "N"
	if lat0 < 0 {
		ns = "S"
		lat0 = -lat0
	}
	ew := "E"
	if lon0 < 0 {
		ew = "W"
		lon0 = -lon0
	}
	return fmt.Sprintf("ESA_WorldCover_10m_2021_v200_%s%02d%s%03d_Map.tif", ns, int(lat0), ew, int(lon0))
}

func worldCoverClassKey(class int) string {
	switch class {
	case 10:
		return "10_tree_cover"
	case 20:
		return "20_shrubland"
	case 30:
		return "30_grassland"
	case 40:
		return "40_cropland"
	case 50:
		return "50_built_up"
	case 60:
		return "60_bare_sparse"
	case 70:
		return "70_snow_ice"
	case 80:
		return "80_permanent_water"
	case 90:
		return "90_herbaceous_wetland"
	case 95:
		return "95_mangrove"
	case 100:
		return "100_moss_lichen"
	default:
		return fmt.Sprintf("%d_unknown", class)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func sortedLandcoverStats(counts map[int]int, total int, keyFn func(int) string) []LandcoverStat {
	if total <= 0 {
		return nil
	}
	if keyFn == nil {
		keyFn = worldCoverClassKey
	}
	out := make([]LandcoverStat, 0, len(counts))
	for class, count := range counts {
		if class == 0 || count == 0 {
			continue
		}
		out = append(out, LandcoverStat{
			Class:  class,
			Key:    keyFn(class),
			Weight: float64(count) / float64(total),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Weight == out[j].Weight {
			return out[i].Class < out[j].Class
		}
		return out[i].Weight > out[j].Weight
	})
	return out
}

func glcCoarseClassKey(class int) string {
	switch class {
	case 10, 11, 12, 20:
		return "40_cropland"
	case 51, 52, 61, 62, 71, 72, 81, 82, 91, 92:
		return "10_tree_cover"
	case 120, 121, 122:
		return "20_shrubland"
	case 130:
		return "30_grassland"
	case 140:
		return "100_moss_lichen"
	case 150, 152, 153, 200, 201, 202:
		return "60_bare_sparse"
	case 181, 182, 183, 184, 186, 187:
		return "90_herbaceous_wetland"
	case 185:
		return "95_mangrove"
	case 190:
		return "50_built_up"
	case 210, 211, 212, 213, 214, 215, 216, 217, 218, 219:
		return "80_permanent_water"
	case 220:
		return "70_snow_ice"
	default:
		return fmt.Sprintf("%d_unknown_glc", class)
	}
}

func glcFCS30DClassKey(class int) string {
	switch class {
	case 10:
		return "rainfed_cropland"
	case 11:
		return "herbaceous_cover_cropland"
	case 12:
		return "tree_or_shrub_cover_cropland"
	case 20:
		return "irrigated_cropland"
	case 51:
		return "open_evergreen_broadleaf_forest"
	case 52:
		return "closed_evergreen_broadleaf_forest"
	case 61:
		return "open_deciduous_broadleaf_forest"
	case 62:
		return "closed_deciduous_broadleaf_forest"
	case 71:
		return "open_evergreen_needleleaf_forest"
	case 72:
		return "closed_evergreen_needleleaf_forest"
	case 81, 82:
		return "deciduous_needleleaf_forest"
	case 91, 92:
		return "mixed_forest"
	case 120, 121, 122:
		return "shrubland"
	case 130:
		return "grassland"
	case 140:
		return "moss_lichen_tundra"
	case 150, 152, 153, 200, 201, 202:
		return "sparse_bare"
	case 181, 183, 184, 187:
		return "swamp_flooded_forest"
	case 182, 186:
		return "herbaceous_wetland_marsh"
	case 185:
		return "mangrove"
	case 190:
		return "impervious_surfaces"
	case 210, 211, 212, 213, 214, 215, 216, 217, 218, 219:
		return "water_body"
	case 220:
		return "permanent_ice_snow"
	default:
		return fmt.Sprintf("%d_unknown_glc", class)
	}
}
