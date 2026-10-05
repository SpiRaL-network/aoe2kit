# AoE2Kit Package API

Go package documentation, emitted by `go doc -all` for every package that
ships in the public release (`release/public-files.json`). This is the
library-level API for importing AoE2Kit directly; `API_REFERENCE.md` covers
the command line, and `api_schemas.json` carries the full nested shape of
every report type.

Regenerate with:

```sh
go run ./cmd/apiref
```

## aoe2kit/cmd/apiref

```go
apiref generates the AoE2Kit API reference by interrogating the kit binary
itself: invocation lines come from kit's own usage output, and output contracts
come from probing read-only commands against real fixtures.

Nothing in the generated reference is hand-asserted. A command that could not be
probed is labeled as such rather than described from assumption.

FUNCTIONS

func CommandResponseTypeHint(commandName string, types map[string]TypeSchema) string
func ExtractSchemas(roots ...string) (map[string]TypeSchema, error)
    ExtractSchemas walks the package tree and returns every exported struct type
    keyed as "package.TypeName".

func MatchResponseType(outputKeys []string, types map[string]TypeSchema) string
    MatchResponseType finds the schema whose top-level JSON names best cover the
    keys a command actually returned. This is the join between the two methods:
    the probe says which type a command emits, the source says its full shape.

func WriteGoDoc(outPath string) error
    WriteGoDoc renders the package-level Go API using the toolchain's own
    documentation, rather than reimplementing it.


TYPES

type CatalogEntry struct {
	Name  string `json:"name"`
	Usage string `json:"usage"`
	Trait string `json:"trait"`
	Input string `json:"input"`
}
    CatalogEntry mirrors one row of the CLI's declared command catalog.

type Command struct {
	Path        []string `json:"path"`
	Name        string   `json:"name"`
	Group       string   `json:"group,omitempty"`
	Usage       string   `json:"usage"`
	Flags       []Flag   `json:"flags,omitempty"`
	Positional  []string `json:"positional,omitempty"`
	Subcommands []string `json:"subcommands,omitempty"`
	Mutates     bool     `json:"mutates"`
	Network     bool     `json:"network"`
	Trait       string   `json:"trait,omitempty"`
	Input       string   `json:"input,omitempty"`

	ProbeStatus  string   `json:"probe_status"`
	ProbeArgs    []string `json:"probe_args,omitempty"`
	ExitCode     int      `json:"probe_exit_code,omitempty"`
	OutputKeys   []string `json:"output_top_level_keys,omitempty"`
	ResponseType string   `json:"response_type,omitempty"`
	Verification string   `json:"verification,omitempty"`
	Method       string   `json:"method,omitempty"`
	ProbeNote    string   `json:"probe_note,omitempty"`
}

type FieldSchema struct {
	Name     string `json:"name"`
	JSON     string `json:"json,omitempty"`
	GoType   string `json:"go_type"`
	Ref      string `json:"ref,omitempty"`
	Optional bool   `json:"optional,omitempty"`
	Embedded bool   `json:"embedded,omitempty"`
	Doc      string `json:"doc,omitempty"`
}

type Fixtures struct {
	Scenario string `json:"scenario,omitempty"`
	Replay   string `json:"replay,omitempty"`
	Dat      string `json:"dat,omitempty"`
	Xsdat    string `json:"xsdat,omitempty"`
	Folder   string `json:"folder,omitempty"`
}

type Flag struct {
	Name     string `json:"name"`
	Value    string `json:"value_hint,omitempty"`
	Required bool   `json:"required,omitempty"`
}

type Reference struct {
	Tool      string    `json:"tool"`
	Version   string    `json:"version"`
	Generated string    `json:"generated_by"`
	Fixtures  Fixtures  `json:"probe_fixtures"`
	CommandsN int       `json:"command_count"`
	ProbedN   int       `json:"probed_count"`
	Commands  []Command `json:"commands"`
}

type SchemaCatalog struct {
	Note  string                `json:"note"`
	Count int                   `json:"type_count"`
	Types map[string]TypeSchema `json:"types"`
}

type TypeSchema struct {
	Package string        `json:"package"`
	Name    string        `json:"name"`
	Doc     string        `json:"doc,omitempty"`
	Fields  []FieldSchema `json:"fields"`
}
```

## aoe2kit/cmd/cba

```go

```

## aoe2kit/cmd/dat

```go

```

## aoe2kit/cmd/kit

```go
CONSTANTS

const (
	TraitReadOnly = "read_only"
	TraitWrites   = "writes"
	TraitNetwork  = "network"
)

TYPES

type CommandSpec struct {
	Name  string `json:"name"`
	Usage string `json:"usage"`
	Trait string `json:"trait"`
	Input string `json:"input"`
}
```

## aoe2kit/cmd/mod

```go

```

## aoe2kit/cmd/scen

```go

```

## aoe2kit/cmd/swatch

```go

```

## aoe2kit/pkg/aifile

```go
package aifile // import "aoe2kit/pkg/aifile"


FUNCTIONS

func IsAIPath(path string) bool
func SaveRegistry(path string, registry Registry) error
func Signature(refs []Reference) string

TYPES

type DiffChange struct {
	Kind   string `json:"kind"`
	File   string `json:"file,omitempty"`
	Before any    `json:"before,omitempty"`
	After  any    `json:"after,omitempty"`
}

type DiffReport struct {
	Before  string       `json:"before"`
	After   string       `json:"after"`
	Same    bool         `json:"same"`
	Changes []DiffChange `json:"changes,omitempty"`
}

func DiffPaths(beforePath, afterPath string) (*DiffReport, error)

type Fingerprint struct {
	Path        string            `json:"path"`
	SHA256      string            `json:"sha256"`
	Method      string            `json:"method"`
	Files       []FingerprintFile `json:"files"`
	RegistryKey string            `json:"registry_key,omitempty"`
	Known       *RegistryEntry    `json:"known,omitempty"`
}

func FingerprintPath(path string) (*Fingerprint, error)

type FingerprintFile struct {
	Path            string `json:"path"`
	RelativePath    string `json:"relative_path"`
	SizeBytes       int64  `json:"size_bytes"`
	NormalizedBytes int    `json:"normalized_bytes"`
	ContentSHA256   string `json:"content_sha256"`
	LineEndingsNote string `json:"line_endings_note,omitempty"`
}

type Group struct {
	Name      string   `json:"name"`
	SourceMod string   `json:"source_mod,omitempty"`
	Files     []string `json:"files"`
	Raw       []string `json:"raw"`
}

func GroupReferences(refs []Reference) []Group

type LintFile struct {
	Path            string `json:"path"`
	RelativePath    string `json:"relative_path"`
	SizeBytes       int64  `json:"size_bytes"`
	NormalizedBytes int    `json:"normalized_bytes"`
	LineEndingsNote string `json:"line_endings_note,omitempty"`
}

type LintIssue struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	File     string `json:"file,omitempty"`
	Message  string `json:"message"`
}

type LintReport struct {
	Path         string                 `json:"path"`
	OK           bool                   `json:"ok"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Files        []LintFile             `json:"files"`
	Issues       []LintIssue            `json:"issues,omitempty"`
	Summary      LintSummary            `json:"summary"`
}

func LintPath(path string) (*LintReport, error)

type LintSummary struct {
	Files  int `json:"files"`
	Errors int `json:"errors"`
	Warns  int `json:"warnings"`
}

type Loadout struct {
	References []Reference `json:"references"`
	Groups     []Group     `json:"groups"`
	Signature  string      `json:"signature"`
	Empty      bool        `json:"empty"`
	Note       string      `json:"note,omitempty"`
}

func NewLoadout(refs []Reference) Loadout

type Reference struct {
	Raw       string `json:"raw"`
	Source    string `json:"source,omitempty"`
	Domain    string `json:"domain,omitempty"`
	File      string `json:"file"`
	BaseName  string `json:"base_name"`
	Extension string `json:"extension"`
	SourceMod string `json:"source_mod,omitempty"`
	Flag      string `json:"flag,omitempty"`
}

func ParseReference(raw string) (Reference, bool)

func ParseReferences(stringsIn []string) []Reference

func (r Reference) Normalized() string

type Registry map[string]RegistryEntry

func LoadRegistry(path string) (Registry, error)

type RegistryEntry struct {
	Name          string   `json:"name"`
	Mod           string   `json:"mod,omitempty"`
	Notes         string   `json:"notes,omitempty"`
	Signature     string   `json:"signature,omitempty"`
	Files         []string `json:"files,omitempty"`
	ContentSHA256 string   `json:"content_sha256,omitempty"`
}
```

## aoe2kit/pkg/aoe2

```go
package aoe2 // import "aoe2kit/pkg/aoe2"


FUNCTIONS

func KindForPath(path string) string

TYPES

type FileInfo struct {
	Path      string `json:"path"`
	Base      string `json:"base"`
	Kind      string `json:"kind"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}

func InspectFile(path string) (FileInfo, error)

type VerificationClaim struct {
	Label             string `json:"label"`
	StructureVerified bool   `json:"structure_verified"`
	EngineVerified    bool   `json:"engine_verified"`
	Note              string `json:"note"`
}

func StructureOKActivationFailed() VerificationClaim

func StructureVerification(ok bool) VerificationClaim
```

## aoe2kit/pkg/aoems

```go
package aoems // import "aoe2kit/pkg/aoems"


CONSTANTS

const (
	DefaultBaseURL = "https://api.ageofempires.com/api"
	DefaultTTL     = 120 * time.Second
)

FUNCTIONS

func DefaultCacheDir() string
func ReplayCachePath(outDir string, gameID string, profileID string) string

TYPES

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	CacheDir   string
	TTL        time.Duration
	UserAgent  string
	Now        func() time.Time
}

func NewClient(cacheDir string) *Client

func (c *Client) FetchReplay(opts ReplayFetchOptions) (*ReplayFetchReport, error)

func (c *Client) PlayerStats(opts PlayerStatsOptions) (*PlayerStatsReport, error)

type PlayerStatsOptions struct {
	ProfileID string
	MatchType int
	Force     bool
}

type PlayerStatsReport struct {
	ProfileID    string          `json:"profile_id"`
	MatchType    int             `json:"match_type"`
	Method       string          `json:"method"`
	Verification string          `json:"verification"`
	URL          string          `json:"url"`
	Cached       bool            `json:"cached"`
	CachePath    string          `json:"cache_path,omitempty"`
	FetchedAt    string          `json:"fetched_at,omitempty"`
	AgeSeconds   int64           `json:"age_seconds,omitempty"`
	StatusCode   int             `json:"status_code,omitempty"`
	User         PlayerStatsUser `json:"user,omitempty"`
	CareerStats  map[string]any  `json:"career_stats,omitempty"`
	MPStatList   any             `json:"mp_stat_list,omitempty"`
	Raw          json.RawMessage `json:"raw,omitempty"`
	Warnings     []string        `json:"warnings,omitempty"`
}

type PlayerStatsUser struct {
	ProfileID            any     `json:"profileId,omitempty"`
	UserName             string  `json:"userName,omitempty"`
	ELO                  any     `json:"elo,omitempty"`
	PlayerStanding       any     `json:"playerStanding,omitempty"`
	PlayerStandingNumber float64 `json:"player_standing_number,omitempty"`
	AvatarURL            string  `json:"avatarUrl,omitempty"`
	IsHuman              *bool   `json:"isHuman,omitempty"`
	MatchReplayAvailable *bool   `json:"matchReplayAvailable,omitempty"`
}

type ReplayFetchAllReport struct {
	GameID       string              `json:"game_id"`
	SeedProfile  string              `json:"seed_profile_id,omitempty"`
	Method       string              `json:"method"`
	Verification string              `json:"verification"`
	OutDir       string              `json:"out_dir"`
	Fetched      []ReplayFetchReport `json:"fetched"`
	Profiles     []string            `json:"profiles"`
	Warnings     []string            `json:"warnings,omitempty"`
}

type ReplayFetchOptions struct {
	GameID    string
	ProfileID string
	OutDir    string
	Force     bool
}

type ReplayFetchReport struct {
	GameID       string   `json:"game_id"`
	ProfileID    string   `json:"profile_id"`
	Method       string   `json:"method"`
	Verification string   `json:"verification"`
	URL          string   `json:"url"`
	Path         string   `json:"path"`
	Cached       bool     `json:"cached"`
	Bytes        int64    `json:"bytes"`
	StatusCode   int      `json:"status_code,omitempty"`
	ContentType  string   `json:"content_type,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}
```

## aoe2kit/pkg/campaign

```go
package campaign // import "aoe2kit/pkg/campaign"


TYPES

type ExtractEntry struct {
	Index       int    `json:"index"`
	Name        string `json:"name"`
	Filename    string `json:"filename"`
	Offset      int    `json:"offset"`
	Size        int    `json:"size"`
	OutPath     string `json:"out_path,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
	Selected    bool   `json:"selected"`
	FullParseOK bool   `json:"full_parse_ok"`
	ParseError  string `json:"parse_error,omitempty"`
}

type ExtractOptions struct {
	OutDir   string
	Scenario int
	Name     string
}

type ExtractReport struct {
	OK             bool           `json:"ok"`
	Path           string         `json:"path"`
	Version        string         `json:"version"`
	DirectoryAt    int            `json:"directory_at"`
	DirectoryCount int            `json:"directory_count"`
	OutDir         string         `json:"out_dir"`
	Entries        []ExtractEntry `json:"entries"`
	Warnings       []string       `json:"warnings,omitempty"`
}

func Extract(data []byte, path string, opts ExtractOptions) (*ExtractReport, error)

func ExtractFile(path string, opts ExtractOptions) (*ExtractReport, error)

type Field struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	PerPlayer bool   `json:"per_player,omitempty"`
	Default   any    `json:"default,omitempty"`
}

type GenerateOptions struct {
	OutDir string
	Prefix string
}

type GenerateReport struct {
	OK             bool     `json:"ok"`
	SpecPath       string   `json:"spec_path"`
	OutDir         string   `json:"out_dir"`
	Name           string   `json:"name"`
	WriterScenario string   `json:"writer_scenario"`
	Files          []string `json:"files"`
	Warnings       []string `json:"warnings,omitempty"`
}

func Generate(spec *Spec, opts GenerateOptions) (*GenerateReport, error)

func GenerateFile(path string, opts GenerateOptions) (*GenerateReport, error)

type Spec struct {
	SchemaVersion  int     `json:"schema_version"`
	Name           string  `json:"name"`
	WriterScenario string  `json:"writer_scenario"`
	Players        int     `json:"players,omitempty"`
	Fields         []Field `json:"fields"`
}

func LoadSpec(path string) (*Spec, error)
```

## aoe2kit/pkg/cba

```go
package cba // import "aoe2kit/pkg/cba"


CONSTANTS

const (
	// DefaultArchiveDelay paces requests. Slow on purpose -- there is no deadline here
	// beyond the retention window, and politeness costs us nothing.
	DefaultArchiveDelay = 4 * time.Second

	// MaxConsecutiveFailures trips the circuit breaker. Repeated failures mean
	// something changed (auth, rate limit, outage); hammering through them is how a
	// well-behaved client turns into a bad one.
	MaxConsecutiveFailures = 5
)
    The archiver exists to win a race we otherwise lose: the match-replay API
    retains a game for roughly two to three weeks, and a laptop that is only
    on some of the time cannot reliably pull inside that window. Run from an
    always-on host, this turns "whatever we happened to catch" into a permanent
    archive.

    It is deliberately unhurried and deliberately bounded. An unattended fetch
    loop against someone else's public API is exactly the thing that becomes
    abusive by accident, so: one request at a time, a real delay between them,
    a hard ceiling that must be stated explicitly, and a circuit breaker that
    stops on repeated failure.

const (
	// MinProdForCombat floors the loss-rate denominator. Without it a player who
	// produced a handful of units posts an absurd preservation ratio and dominates
	// the axis on noise.
	MinProdForCombat = 100
)
    CBA skill is multi-axis and the axes are orthogonal -- never collapse them
    to one number. Every axis is scored as CONTRIBUTION SHARE WITHIN A GAME,
    so a rider on a strong team earns little regardless of the result. That is
    what makes stacking self-defeating WITHOUT needing manual stacker tags.

    "Genuinely good = multiple positive correlation" (chrae): independent axes
    AGREE.

        ECO     share of your team's production channels, vs an equal share
        TEMPO   how early you open the floodgates, vs the others in that game
        COMBAT  army preservation, normalised within your OWN TEAM
        CONDUCT quit behaviour, side tendency (descriptive)

const (
	LadderToken       = "CBA_=REQUIEM=_V292"
	RequiemTriggerMin = 2700
	RequiemTriggerMax = 3400
)
    The ladder admits exactly one scenario: the community's official CBA Requiem
    V292.

    Identity is the EMBEDDED FILENAME TOKEN plus a trigger-count band --
    deliberately not the lobby name (hosts rename lobbies freely, so lobby text
    is worthless as identity) and deliberately not the trigger-graph hash (that
    identifies an EXACT host copy, and the same community V292 ships as at least
    three host copies with different graphs). The token is community identity;
    the band rejects same-named forks and test builds.

const (
	ExtractWorkerMiB  = 1100
	RegistryWorkerMiB = 300
)
    Per-worker memory budgets, in MiB, from measured peak RSS of a single
    replay:

        extraction (BuildPerformance): ~860 MiB  -- inflated replay + profile + events +
                                                    camera + checksum series + razes
        classification (registry):     ~207 MiB  -- inflated replay + player profile only

    Budgets sit above the measurement because replay size varies and Go returns
    freed memory to the OS lazily.

const MinCivSamplesForBaseline = 8
    MinCivSamplesForBaseline is how many scoreable rows a civ needs before
    its baseline is trusted. Below this the civ is left UNADJUSTED rather than
    corrected with a noisy number -- and never silently folded into a global
    average, which would quietly move rare-civ players for no measured reason.


VARIABLES

var KnownAnchors = []string{
	"Monkey Boy", "Cengo", "man23457689", "[xCs]Bek", "[xCs]Vangelis", "nW | Sgt.Pepper",
}
    KnownAnchors are players chrae vouches for as genuinely strong, used to
    sanity-check that the model agrees with human ground truth. Ratings are NOT
    pinned to them here -- the axes are contribution-based and must stand on
    their own.

var KnownStackers = []string{"Aster"}
    KnownStackers are players chrae has observed stacking lobbies. Stacking
    is ORTHOGONAL to skill (man23457689 is elite and stacks); this tag exists
    to validate that the contribution axes neutralise a stacker's inflated win
    rate, not to penalise them.


FUNCTIONS

func AppendManifest(path string, records []ArchiveRecord) error
    AppendManifest appends records. Append-only by design: the manifest is
    the record of what we asked the API for, and rewriting it would lose that
    history.

func Archive(queue []QueueItem, seen map[string]bool, client *aoems.Client, opts ArchiveOptions) (ArchiveResult, []ArchiveRecord)
    Archive fetches queued matches, keeps the CBA ones, and records every
    outcome.

    Classification happens AFTER download because scenario identity lives inside
    the replay -- there is no way to know a match is CBA without its bytes.
    Classification is the light path (open + header token + trigger count),
    not the heavy extraction, so it is cheap enough to run on a shared host.

func CivRazeArchetype(civID int) string
func ExportSQL(w io.Writer, c *Corpus, axes []PlayerAxes, names map[int]string) error
    ExportSQL emits a portable SQL script for the ladder site. Text rather
    than a binary database file on purpose: writing SQLite directly would
    require a CGO/pure-Go driver dependency, and AoE2Kit has none. Piping
    through the sqlite3 CLI (present both locally and on the host) keeps the
    kit dependency-free, and the same script loads into MySQL if the site ever
    outgrows a file.

        kit cba export --sql > ladder.sql && sqlite3 ladder.db < ladder.sql

    The script is idempotent -- it drops and rebuilds every table, because
    the ladder is regenerated wholesale from replays rather than updated
    incrementally.

func GameIDForPath(p string) string
    GameIDForPath derives a game id from the replay filename, preferring the
    Microsoft match id when the name carries one.

    The same match reaches us under completely different filenames -- a local
    save ("037__MP Replay v101...") and an MS-API pull ("492398692.zip") -- so a
    bare basename is NOT a stable identity: re-pulling a match under a new local
    name would mint a second id for a game already in the corpus. The MS id is
    stable across sources and re-pulls, so it wins whenever it is present.

func HasMSGameID(gameID string) bool
    HasMSGameID reports whether an id is a Microsoft match id rather than a
    local filename. Used to pick the canonical copy when one match arrives from
    both sources.

func LoadManifest(path string) (map[string]bool, error)
    LoadManifest returns the set of already-processed game ids, so a re-run
    never re-requests a match the API has already answered for -- including
    rejects.

func WriteRegistry(w io.Writer, reg *Registry) error
    WriteRegistry persists the admission decisions, including rejections and
    their reasons -- a registry that recorded only what it admitted could not be
    audited.


TYPES

type ArchiveOptions struct {
	// OutDir receives admitted CBA replays.
	OutDir string
	// RejectDir receives everything else. Non-CBA downloads are MOVED here rather than
	// deleted -- the archiver should not be in the business of irreversible cleanup,
	// and a reject pile is easy to inspect or purge by hand.
	RejectDir string
	// Max is a hard ceiling on fetches this run. Required: there is no safe default
	// for "how much of someone else's API should I pull unattended".
	Max int
	// Delay between requests (0 = DefaultArchiveDelay).
	Delay time.Duration
	// Progress is called after each item.
	Progress func(rec ArchiveRecord)
}
    ArchiveOptions configures a run.

type ArchiveRecord struct {
	GameID    string `json:"game_id"`
	ProfileID string `json:"profile_id"`
	FetchedAt string `json:"fetched_at"`
	Bytes     int64  `json:"bytes,omitempty"`
	Path      string `json:"path,omitempty"`
	Admitted  bool   `json:"admitted"`
	Token     string `json:"token,omitempty"`
	Triggers  int    `json:"trigger_count,omitempty"`
	Reason    string `json:"reason"`
	Note      string `json:"note,omitempty"`
}
    ArchiveRecord is the durable outcome for one queue item. Written for rejects
    too: without them the archiver would re-download every non-CBA game on every
    run.

type ArchiveResult struct {
	Considered int    `json:"considered"`
	Fetched    int    `json:"fetched"`
	Admitted   int    `json:"admitted"`
	Rejected   int    `json:"rejected"`
	Skipped    int    `json:"skipped"`
	Failed     int    `json:"failed"`
	StoppedBy  string `json:"stopped_by,omitempty"`
}
    ArchiveResult summarises a run.

type AxesOptions struct {
	// MinGames drops players too thin to score (default 3).
	MinGames int
	// Names maps profile id -> display name.
	Names map[int]string
	// Anchors / Stackers override the package defaults when non-nil.
	Anchors  []string
	Stackers []string
}
    AxesOptions configures scoring.

type BalanceEvidence struct {
	Version    string `json:"version"`
	Kind       string `json:"kind"`
	Attribute  int    `json:"attribute,omitempty"`
	Amounts    []int  `json:"amounts,omitempty"`
	Counts     []int  `json:"counts,omitempty"`
	Source     string `json:"source"`
	Confidence string `json:"confidence"`
	Notes      string `json:"notes,omitempty"`
}

type BalanceMissing struct {
	Version string `json:"version"`
	CivID   int    `json:"civ_id,omitempty"`
	CivName string `json:"civ_name,omitempty"`
	Reason  string `json:"reason"`
}

type BalanceReport struct {
	Method       string            `json:"method"`
	Verification string            `json:"verification"`
	Summary      BalanceSummary    `json:"summary"`
	Rows         []BalanceRow      `json:"rows"`
	Evidence     []BalanceEvidence `json:"evidence,omitempty"`
	Missing      []BalanceMissing  `json:"missing,omitempty"`
	Warnings     []string          `json:"warnings,omitempty"`
}

func BuildBalance() *BalanceReport

type BalanceRow struct {
	Version          string   `json:"version"`
	CivID            int      `json:"civ_id,omitempty"`
	CivName          string   `json:"civ_name"`
	RazesToVillager  *int     `json:"razes_to_villager,omitempty"`
	CastleKills      *int     `json:"castle_kills,omitempty"`
	ImperialKills    *int     `json:"imperial_kills,omitempty"`
	UnitCount        *int     `json:"unit_count,omitempty"`
	SpawnSeconds     *int     `json:"spawn_seconds,omitempty"`
	StableElephants  *int     `json:"stable_elephants,omitempty"`
	BattleElephants  *int     `json:"battle_elephants,omitempty"`
	Notes            string   `json:"notes,omitempty"`
	Sources          []string `json:"sources"`
	Coverage         string   `json:"coverage"`
	PreferredForV292 bool     `json:"preferred_for_v292,omitempty"`
}

type BalanceSummary struct {
	Rows            int `json:"rows"`
	CivRows         int `json:"civ_rows"`
	VersionRows     int `json:"version_rows"`
	TriggerEvidence int `json:"trigger_evidence"`
	MissingRows     int `json:"missing_rows"`
}

type Corpus struct {
	Games []GameRecord
}
    Corpus is the durable store behind the CBA ladder: one GameRecord per
    ingested replay, persisted as JSONL. JSONL rather than SQLite deliberately
    -- AoE2Kit has no external dependencies and that portability is worth more
    than ad-hoc SQL for a corpus this size (hundreds of games, thousands of
    rows, all loaded and grouped in memory anyway).

func LoadCorpus(path string) (*Corpus, error)
    LoadCorpus reads a JSONL corpus. A missing file is an empty corpus, not an
    error -- ingest is expected to bootstrap it.

func (c *Corpus) Has(gameID string) bool
    Has reports whether a game is already ingested, so ingest is idempotent and
    a re-run over the whole replay tree costs nothing for games already present.

func (c *Corpus) Rows() []PerformanceRow
    Rows flattens every performance row across the corpus, stamping GameID
    onto rows that lack it so downstream grouping never has to carry the parent
    record.

func (c *Corpus) Save(path string) error
    Save writes the corpus atomically -- a torn corpus file would silently
    truncate the ladder, and the failure mode (fewer games, plausible numbers)
    is invisible.

func (c *Corpus) Upsert(g GameRecord)
    Upsert replaces an existing record with the same GameID, else appends.

type DoctrinePlayerRow struct {
	Slot                int      `json:"slot"`
	ProfileID           int      `json:"profile_id,omitempty"`
	Name                string   `json:"name,omitempty"`
	Team                string   `json:"team,omitempty"`
	CivID               int      `json:"civ_id,omitempty"`
	Civ                 string   `json:"civ,omitempty"`
	Won                 *int     `json:"won,omitempty"`
	Duty                string   `json:"duty"`
	DutyConfidence      string   `json:"duty_confidence"`
	PhaseRead           string   `json:"phase_read"`
	OwnFirstVillS       *int     `json:"own_first_vill_s,omitempty"`
	TeamFirstVillS      *int     `json:"team_first_vill_s,omitempty"`
	OwnRazes            *int     `json:"own_razes,omitempty"`
	TeamRazes           *int     `json:"team_razes,omitempty"`
	FirstPressureTime   string   `json:"first_pressure_time,omitempty"`
	FirstRazeCandidate  string   `json:"first_raze_candidate,omitempty"`
	SetPlayFinishes     int      `json:"set_play_finishes,omitempty"`
	FirstProdBuildS     *int     `json:"first_prod_build_s,omitempty"`
	ProdBuildings       *int     `json:"prod_buildings,omitempty"`
	UnitsProduced       *int     `json:"units_produced,omitempty"`
	UnitsLost           *int     `json:"units_lost,omitempty"`
	ProgressionSignal   string   `json:"progression_signal,omitempty"`
	CombatProxy         string   `json:"combat_proxy,omitempty"`
	MetricConfidence    string   `json:"metric_confidence"`
	AttributionWarnings []string `json:"attribution_warnings,omitempty"`
}

type DoctrineReport struct {
	Path         string              `json:"path,omitempty"`
	Method       string              `json:"method"`
	Verification string              `json:"verification"`
	Summary      DoctrineSummary     `json:"summary"`
	Players      []DoctrinePlayerRow `json:"players,omitempty"`
	Warnings     []string            `json:"warnings,omitempty"`
}

func BuildDoctrine(path string) (*DoctrineReport, error)

type DoctrineSummary struct {
	Players                  int    `json:"players"`
	Duration                 string `json:"duration,omitempty"`
	WinnerKnown              bool   `json:"winner_known"`
	RazeCandidates           int    `json:"raze_candidates"`
	SetPlayCandidates        int    `json:"set_play_candidates"`
	PlayersWithOwnVill       int    `json:"players_with_own_vill"`
	PlayersWithTeamVill      int    `json:"players_with_team_vill"`
	PlayersWithDuty          int    `json:"players_with_duty"`
	DirectKillAttribution    string `json:"direct_kill_attribution"`
	DirectRazeAttribution    string `json:"direct_raze_attribution"`
	ReplayPhaseBoundaryBasis string `json:"replay_phase_boundary_basis"`
}

type EngineAttributeRow struct {
	Attribute      int    `json:"attribute"`
	Name           string `json:"name"`
	Semantic       string `json:"semantic"`
	ConditionType  int    `json:"condition_type"`
	ConditionName  string `json:"condition_name"`
	Quantities     []int  `json:"quantities"`
	Players        []int  `json:"players"`
	ConditionCount int    `json:"condition_count"`
	Notes          string `json:"notes,omitempty"`
}

type GameRecord struct {
	GameID     string           `json:"game_id"`
	Path       string           `json:"path"`
	Scenario   string           `json:"scenario,omitempty"`
	DurationS  int              `json:"duration_s,omitempty"`
	IngestedBy string           `json:"ingested_by,omitempty"`
	Rows       []PerformanceRow `json:"performances"`
	Warnings   []string         `json:"warnings,omitempty"`
}
    GameRecord is one ingested replay: the identity we matched it by, plus the
    extractor's per-player performance rows verbatim. The rows are NOT reshaped
    on ingest -- the extractor's schema is the corpus schema, so a change there
    flows through without a migration.

type IngestOptions struct {
	// Force re-extracts games already present instead of skipping them.
	Force bool
	// Limit caps how many NEW games are ingested this run (0 = no cap).
	Limit int
	// Workers sets extraction concurrency. 0 means "fit it in available memory" --
	// see workersFor. Extraction peaks near 900 MiB per replay, so this is bounded by
	// RAM, not by core count; setting it to NumCPU on a small machine will swap-storm.
	Workers int
	// Progress, if set, is called per replay with the outcome. It may be invoked from
	// multiple goroutines, so implementations must be safe to call concurrently.
	Progress func(path, status string, err error)
}
    IngestOptions controls a corpus build.

type IngestResult struct {
	Scanned int `json:"scanned"`
	Added   int `json:"added"`
	Updated int `json:"updated"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
	// Workers records the concurrency actually used. Reported because it is chosen
	// from available memory rather than fixed, and an invisible default is exactly
	// how this job once ate a machine's swap.
	Workers int `json:"workers"`
}
    IngestResult reports what a run did.

func Ingest(c *Corpus, paths []string, opts IngestOptions) IngestResult
    Ingest walks replay paths, extracts performance rows, and folds them into
    the corpus. Extraction failure on one replay is recorded and skipped -- a
    single unreadable file must never abort a long corpus build.

type PerformanceEvent struct {
	Kind             string `json:"kind"`
	TimeMS           int    `json:"time_ms"`
	Time             string `json:"time"`
	PlayerID         int    `json:"player_id"`
	PlayerLabel      string `json:"player_label,omitempty"`
	Team             string `json:"team,omitempty"`
	Count            int    `json:"count,omitempty"`
	UnitID           int    `json:"unit_id,omitempty"`
	UnitName         string `json:"unit_name,omitempty"`
	BuildingID       int    `json:"building_id,omitempty"`
	BuildingName     string `json:"building_name,omitempty"`
	TargetID         int    `json:"target_id,omitempty"`
	TargetClass      string `json:"target_class,omitempty"`
	TargetOwnerID    int    `json:"target_owner_id,omitempty"`
	TargetOwnerLabel string `json:"target_owner_label,omitempty"`
	TargetUnitID     int    `json:"target_unit_id,omitempty"`
	TargetUnitName   string `json:"target_unit_name,omitempty"`
	Source           string `json:"source"`
	Confidence       string `json:"confidence"`
}

type PerformanceReport struct {
	Path         string             `json:"path,omitempty"`
	Method       string             `json:"method"`
	Verification string             `json:"verification"`
	Summary      PerformanceSummary `json:"summary"`
	Rows         []PerformanceRow   `json:"performances"`
	Events       []PerformanceEvent `json:"events,omitempty"`
	Warnings     []string           `json:"warnings,omitempty"`
}

func BuildPerformance(path string) (*PerformanceReport, error)

type PerformanceRow struct {
	GameID              string   `json:"game_id,omitempty"`
	ProfileID           int      `json:"profile_id,omitempty"`
	Slot                int      `json:"slot"`
	Team                string   `json:"team,omitempty"`
	Civ                 string   `json:"civ,omitempty"`
	Won                 *int     `json:"won"`
	APM                 *float64 `json:"apm"`
	ControlGroups       *int     `json:"control_groups"`
	CGSwitches          *int     `json:"cg_switches"`
	Spatial             *int     `json:"spatial"`
	Chat                *int     `json:"chat"`
	Taunts              *int     `json:"taunts"`
	Razes               *int     `json:"razes"`
	VillagersGained     *int     `json:"villagers_gained"`
	OwnFirstVillS       *int     `json:"own_first_vill_s"`
	OwnVills            *int     `json:"own_vills"`
	OwnRazes            *int     `json:"own_razes"`
	OwnFirstRazeS       *int     `json:"own_first_raze_s"`
	TeamFirstVillS      *int     `json:"team_first_vill_s"`
	TeamVills           *int     `json:"team_vills"`
	TeamRazes           *int     `json:"team_razes"`
	TeamFirstRazeS      *int     `json:"team_first_raze_s"`
	ProdBuildings       *int     `json:"prod_buildings"`
	ProdBarracks        *int     `json:"prod_barracks"`
	ProdRange           *int     `json:"prod_range"`
	ProdStable          *int     `json:"prod_stable"`
	ProdSiege           *int     `json:"prod_siege"`
	ProdCastle          *int     `json:"prod_castle"`
	FirstProdBuildS     *int     `json:"first_prod_build_s"`
	DefensiveBuilds     *int     `json:"defensive_builds"`
	ViewlockEvents      *int     `json:"viewlock_events"`
	ViewlockDist        *int     `json:"viewlock_dist"`
	ViewlockJumps       *int     `json:"viewlock_jumps"`
	VocabMove           *float64 `json:"vocab_move"`
	VocabPatrol         *float64 `json:"vocab_patrol"`
	VocabStance         *float64 `json:"vocab_stance"`
	VocabQueue          *float64 `json:"vocab_queue"`
	Resigned            *int     `json:"resigned"`
	ResignTimeS         *int     `json:"resign_time_s"`
	UnitsProduced       *int     `json:"units_produced"`
	UnitsLost           *int     `json:"units_lost"`
	UnitsKilled         *int     `json:"units_killed"`
	MetricConfidence    string   `json:"metric_confidence"`
	AttributionWarnings []string `json:"attribution_warnings,omitempty"`
}

type PerformanceSummary struct {
	Players            int    `json:"players"`
	DurationMS         int    `json:"duration_ms,omitempty"`
	Duration           string `json:"duration,omitempty"`
	WinnerKnown        bool   `json:"winner_known"`
	RowsWithProfileID  int    `json:"rows_with_profile_id"`
	RazeCandidates     int    `json:"raze_candidates"`
	SetPlayCandidates  int    `json:"set_play_candidates"`
	ChecksumSeriesRows int    `json:"checksum_series_rows"`
	MetricEvents       int    `json:"metric_events"`
}

type PhaseAnchor struct {
	Phase              string  `json:"phase"`
	TimeMS             int     `json:"time_ms"`
	Time               string  `json:"time"`
	KillAnchor         int     `json:"kill_anchor"`
	Signal             string  `json:"signal"`
	UnitID             int     `json:"unit_id,omitempty"`
	UnitName           string  `json:"unit_name,omitempty"`
	ObjectCountDelta   int64   `json:"object_count_delta,omitempty"`
	UnitTypeSumDelta   int64   `json:"unit_type_sum_delta,omitempty"`
	AverageAddedTypeID float64 `json:"average_added_type_id,omitempty"`
	Confidence         string  `json:"confidence"`
	Evidence           string  `json:"evidence"`
}

type PhaseCivFact struct {
	CivID             int    `json:"civ_id"`
	CivName           string `json:"civ_name"`
	CastleThreshold   int    `json:"castle_threshold_kills"`
	ImperialThreshold int    `json:"imperial_threshold_kills"`
	RazesToVillager   int    `json:"razes_to_villager,omitempty"`
	CastleEvidence    string `json:"castle_evidence"`
	ImperialEvidence  string `json:"imperial_evidence"`
	RazeEvidence      string `json:"raze_evidence,omitempty"`
	Confidence        string `json:"confidence"`
}

type PhaseConditionGroup struct {
	ActivatorTriggerID int             `json:"activator_trigger_id"`
	Terms              []PhaseTechTerm `json:"terms"`
	ResolvedCivIDs     []int           `json:"resolved_civ_ids,omitempty"`
	ResolvedCivNames   []string        `json:"resolved_civ_names,omitempty"`
	Confidence         string          `json:"confidence"`
}

type PhaseFactsReport struct {
	Path         string            `json:"path,omitempty"`
	DatPath      string            `json:"dat_path,omitempty"`
	Method       string            `json:"method"`
	Verification string            `json:"verification"`
	Summary      PhaseFactsSummary `json:"summary"`
	Rungs        []PhaseRungRow    `json:"rungs,omitempty"`
	Civs         []PhaseCivFact    `json:"civs,omitempty"`
	Warnings     []string          `json:"warnings,omitempty"`
}

func BuildPhaseFacts(path, datPath string) (*PhaseFactsReport, error)

type PhaseFactsSummary struct {
	TriggerCount      int  `json:"trigger_count"`
	CastleRungs       int  `json:"castle_rungs"`
	ImperialRungs     int  `json:"imperial_rungs"`
	ActivatorTriggers int  `json:"activator_triggers"`
	ConditionGroups   int  `json:"condition_groups"`
	CivFacts          int  `json:"civ_facts"`
	DefaultInferred   int  `json:"default_inferred"`
	RazeFactsJoined   int  `json:"raze_facts_joined"`
	LaneSymmetryOK    bool `json:"lane_symmetry_ok"`
	UnresolvedGroups  int  `json:"unresolved_groups"`
	MultiCivGroups    int  `json:"multi_civ_groups"`
}

type PhaseRungRow struct {
	Age             string                `json:"age"`
	ThresholdKills  int                   `json:"threshold_kills"`
	Player          int                   `json:"player"`
	RungTriggerID   int                   `json:"rung_trigger_id"`
	ActivatorIDs    []int                 `json:"activator_trigger_ids,omitempty"`
	ConditionGroups []PhaseConditionGroup `json:"condition_groups,omitempty"`
	Confidence      string                `json:"confidence"`
}

type PhaseTechTerm struct {
	TechnologyID   int    `json:"technology_id"`
	TechnologyName string `json:"technology_name,omitempty"`
	CivID          int    `json:"civ_id,omitempty"`
	CivName        string `json:"civ_name,omitempty"`
	Inverted       bool   `json:"inverted,omitempty"`
}

type PhaseTimelinePlayer struct {
	PlayerID               int          `json:"player_id"`
	PlayerLabel            string       `json:"player_label"`
	PlayerName             string       `json:"player_name,omitempty"`
	CivID                  int          `json:"civ_id,omitempty"`
	CivName                string       `json:"civ_name,omitempty"`
	CastleThresholdKills   int          `json:"castle_threshold_kills,omitempty"`
	ImperialThresholdKills int          `json:"imperial_threshold_kills,omitempty"`
	RazesToVillager        int          `json:"razes_to_villager,omitempty"`
	Castle                 *PhaseAnchor `json:"castle,omitempty"`
	Imperial               *PhaseAnchor `json:"imperial,omitempty"`
	EarlySpawnUnitID       int          `json:"early_spawn_unit_id,omitempty"`
	EarlySpawnUnitName     string       `json:"early_spawn_unit_name,omitempty"`
	EarlySpawnConfidence   string       `json:"early_spawn_confidence,omitempty"`
	DetectionWarnings      []string     `json:"detection_warnings,omitempty"`
}

type PhaseTimelineReport struct {
	Path         string                `json:"path,omitempty"`
	DatPath      string                `json:"dat_path,omitempty"`
	Method       string                `json:"method"`
	Verification string                `json:"verification"`
	Summary      PhaseTimelineSummary  `json:"summary"`
	Players      []PhaseTimelinePlayer `json:"players,omitempty"`
	Warnings     []string              `json:"warnings,omitempty"`
}

func BuildPhaseTimeline(path, datPath string) (*PhaseTimelineReport, error)

type PhaseTimelineSummary struct {
	Players          int    `json:"players"`
	CastleObserved   int    `json:"castle_observed"`
	ImperialObserved int    `json:"imperial_observed"`
	PhaseFactsCivs   int    `json:"phase_facts_civs"`
	SyncDeltas       int    `json:"sync_deltas"`
	DurationMS       int    `json:"duration_ms"`
	Duration         string `json:"duration"`
	Confidence       string `json:"confidence"`
}

type PlayerAxes struct {
	ProfileID int     `json:"profile_id"`
	Name      string  `json:"name,omitempty"`
	Games     int     `json:"games"`
	Eco       float64 `json:"eco,omitempty"`
	HasEco    bool    `json:"-"`
	Tempo     float64 `json:"tempo,omitempty"`
	HasTempo  bool    `json:"-"`
	Combat    float64 `json:"combat,omitempty"`
	HasCombat bool    `json:"-"`
	Verdict   string  `json:"verdict,omitempty"`

	ProdAvg    float64 `json:"prod_avg"`
	APM        float64 `json:"apm,omitempty"`
	WinPct     int     `json:"win_pct"`
	Decided    int     `json:"decided"`
	EarlyQuit  int     `json:"early_quit_pct"`
	EarlySlot  int     `json:"early_slot_pct"`
	IsAnchor   bool    `json:"anchor,omitempty"`
	IsStacker  bool    `json:"stacker,omitempty"`
	EcoSamples int     `json:"eco_samples"`
	CmbSamples int     `json:"combat_samples"`
	// CombatCivAdjusted counts how many combat samples had a trusted civ baseline
	// applied. Reported so a profile never hides that some rows went uncorrected.
	CombatCivAdjusted int `json:"combat_civ_adjusted"`
}
    PlayerAxes is one player's multi-axis profile.

func ComputeAxes(c *Corpus, opts AxesOptions) []PlayerAxes
    ComputeAxes scores every player in the corpus.

type ProductionFirstSeen struct {
	PlayerID    int    `json:"player_id"`
	PlayerLabel string `json:"player_label,omitempty"`
	TimeMS      int    `json:"time_ms"`
	Time        string `json:"time"`
	UnitID      int    `json:"unit_id"`
	UnitName    string `json:"unit_name,omitempty"`
	Amount      int    `json:"amount,omitempty"`
	BuildingID  int    `json:"building_id,omitempty"`
	Signal      string `json:"signal,omitempty"`
	Confidence  string `json:"confidence"`
}

type ProgressionPlayerSummary struct {
	PlayerID             int    `json:"player_id"`
	Label                string `json:"label"`
	Name                 string `json:"name,omitempty"`
	CivID                int    `json:"civ_id,omitempty"`
	CivName              string `json:"civ_name,omitempty"`
	FirstProductionTime  string `json:"first_production_time,omitempty"`
	FirstProductionUnit  string `json:"first_production_unit,omitempty"`
	FirstImperialProxy   string `json:"first_imperial_proxy,omitempty"`
	FirstImperialProxyAt string `json:"first_imperial_proxy_at,omitempty"`
	FeudalProductionFlag bool   `json:"feudal_production_flag,omitempty"`
	FeudalProductionUnit string `json:"feudal_production_unit,omitempty"`
	FeudalProductionAt   string `json:"feudal_production_at,omitempty"`
	ProductionUnitTypes  int    `json:"production_unit_types"`
	ResearchTechnologies int    `json:"research_technologies"`
	Confidence           string `json:"confidence"`
}

type ProgressionReport struct {
	Path         string                     `json:"path,omitempty"`
	Method       string                     `json:"method"`
	Verification string                     `json:"verification"`
	Summary      ProgressionSummary         `json:"summary"`
	Players      []ProgressionPlayerSummary `json:"players,omitempty"`
	Production   []ProductionFirstSeen      `json:"production_first_seen,omitempty"`
	Research     []ResearchFirstSeen        `json:"research_first_seen,omitempty"`
	Warnings     []string                   `json:"warnings,omitempty"`
}

func BuildProgression(path string) (*ProgressionReport, error)

type ProgressionSummary struct {
	ProductionEvents      int `json:"production_events"`
	ResearchEvents        int `json:"research_events"`
	PlayersWithProduction int `json:"players_with_production"`
	PlayersWithResearch   int `json:"players_with_research"`
	FeudalProductionFlags int `json:"feudal_production_flags"`
	ImperialProxySignals  int `json:"imperial_proxy_signals"`
}

type QueueItem struct {
	GameID    string `json:"game_id"`
	ProfileID string `json:"profile_id"`
	// Note is free-text provenance from discovery (e.g. which player's history it came
	// from). Carried through to the manifest so an archived game can be traced back.
	Note string `json:"note,omitempty"`
}
    QueueItem is one match to archive. Discovery happens elsewhere -- Cloudflare
    blocks non-browser clients from the match-listing site, so ids are produced
    by a browser session and handed here as data.

func LoadQueue(path string) ([]QueueItem, error)
    LoadQueue reads discovery output: either a JSON array or one JSON object per
    line.

type RazeCandidate struct {
	TimeMS            int     `json:"time_ms"`
	Time              string  `json:"time"`
	PlayerID          int     `json:"player_id"`
	PlayerLabel       string  `json:"player_label,omitempty"`
	VillagerObjectID  int     `json:"villager_object_id"`
	VillagerUnitID    int     `json:"villager_unit_id,omitempty"`
	VillagerUnitName  string  `json:"villager_unit_name,omitempty"`
	TargetID          int     `json:"target_id"`
	TargetClass       string  `json:"target_class,omitempty"`
	TargetOwnerID     int     `json:"target_owner_id,omitempty"`
	TargetOwnerLabel  string  `json:"target_owner_label,omitempty"`
	TargetUnitID      int     `json:"target_unit_id,omitempty"`
	TargetUnitName    string  `json:"target_unit_name,omitempty"`
	TargetX           float64 `json:"target_x,omitempty"`
	TargetY           float64 `json:"target_y,omitempty"`
	LastAttackTimeMS  int     `json:"last_attack_time_ms"`
	LastAttackTime    string  `json:"last_attack_time"`
	DeltaMS           int     `json:"delta_ms"`
	Delta             string  `json:"delta"`
	DistinctAttackers int     `json:"distinct_attackers"`
	Confidence        string  `json:"confidence"`
	Reason            string  `json:"reason"`
}

type RazePlayerSummary struct {
	PlayerID             int    `json:"player_id"`
	Label                string `json:"label"`
	Name                 string `json:"name,omitempty"`
	CivID                int    `json:"civ_id,omitempty"`
	CivName              string `json:"civ_name,omitempty"`
	RazeArchetype        string `json:"raze_archetype"`
	FirstPressureTime    string `json:"first_pressure_time,omitempty"`
	FirstPressureTarget  int    `json:"first_pressure_target,omitempty"`
	PressureOrder        int    `json:"pressure_order,omitempty"`
	PressureArchetypeFit string `json:"pressure_archetype_fit,omitempty"`
	FirstRazeCandidate   string `json:"first_raze_candidate,omitempty"`
	FirstRazeTarget      int    `json:"first_raze_target,omitempty"`
	SetPlayFinishes      int    `json:"set_play_finishes"`
	RazeOrder            int    `json:"raze_order,omitempty"`
	ArchetypeFit         string `json:"archetype_fit,omitempty"`
	Confidence           string `json:"confidence"`
}

type RazePressureAttacker struct {
	PlayerID   int    `json:"player_id"`
	PlayerName string `json:"player_name,omitempty"`
	Events     int    `json:"events"`
}

type RazePressurePlayer struct {
	PlayerID              int    `json:"player_id"`
	PlayerLabel           string `json:"player_label"`
	PlayerName            string `json:"player_name,omitempty"`
	PressureEvents        int    `json:"pressure_events"`
	PrimaryPressureEvents int    `json:"primary_pressure_events"`
	Targets               int    `json:"targets"`
	PrimaryTargets        int    `json:"primary_targets"`
	NearbyLossTargets     int    `json:"nearby_loss_targets"`
	NetObjectsLostNearby  int    `json:"net_objects_lost_nearby"`
}

type RazePressureReport struct {
	Path         string               `json:"path,omitempty"`
	Method       string               `json:"method"`
	Verification string               `json:"verification"`
	Summary      RazePressureSummary  `json:"summary"`
	Players      []RazePressurePlayer `json:"players,omitempty"`
	Targets      []RazePressureTarget `json:"targets,omitempty"`
	Warnings     []string             `json:"warnings,omitempty"`
}

func BuildRazePressure(path string) (*RazePressureReport, error)

type RazePressureSummary struct {
	Targets              int `json:"targets"`
	PrimaryTargets       int `json:"primary_targets"`
	GateTargets          int `json:"gate_targets"`
	CastleTargets        int `json:"castle_targets"`
	WallTargets          int `json:"wall_targets"`
	OtherBuildingTargets int `json:"other_building_targets"`
	PressureEvents       int `json:"pressure_events"`
	NearbyLossPulses     int `json:"nearby_loss_pulses"`
	NetObjectsLostNearby int `json:"net_objects_lost_nearby"`
	PlayersWithPressure  int `json:"players_with_pressure"`
}

type RazePressureTarget struct {
	TargetID             int                    `json:"target_id"`
	OwnerID              int                    `json:"owner_id,omitempty"`
	OwnerLabel           string                 `json:"owner_label,omitempty"`
	OwnerName            string                 `json:"owner_name,omitempty"`
	UnitID               int                    `json:"unit_id,omitempty"`
	UnitName             string                 `json:"unit_name,omitempty"`
	Kind                 string                 `json:"kind"`
	Primary              bool                   `json:"primary"`
	X                    float64                `json:"x,omitempty"`
	Y                    float64                `json:"y,omitempty"`
	FirstPressureTimeMS  int                    `json:"first_pressure_time_ms,omitempty"`
	FirstPressureTime    string                 `json:"first_pressure_time,omitempty"`
	LastPressureTimeMS   int                    `json:"last_pressure_time_ms,omitempty"`
	LastPressureTime     string                 `json:"last_pressure_time,omitempty"`
	PressureEvents       int                    `json:"pressure_events"`
	NearbyLossPulses     int                    `json:"nearby_loss_pulses"`
	NetObjectsLostNearby int                    `json:"net_objects_lost_nearby"`
	Attackers            []RazePressureAttacker `json:"attackers,omitempty"`
	Confidence           string                 `json:"confidence"`
}

type RazeReport struct {
	Path         string              `json:"path,omitempty"`
	Method       string              `json:"method"`
	Verification string              `json:"verification"`
	Summary      RazeSummary         `json:"summary"`
	Players      []RazePlayerSummary `json:"players,omitempty"`
	Targets      []TargetPressure    `json:"targets,omitempty"`
	Events       []RazeCandidate     `json:"events,omitempty"`
	SetPlays     []SetPlayCandidate  `json:"set_plays,omitempty"`
	Warnings     []string            `json:"warnings,omitempty"`
}

func BuildRazes(path string) (*RazeReport, error)

type RazeSummary struct {
	BuildingTargetEvents int `json:"building_target_events"`
	PressuredTargets     int `json:"pressured_targets"`
	MultiPlayerTargets   int `json:"multi_player_targets"`
	RazeCandidates       int `json:"raze_candidates"`
	SetPlayCandidates    int `json:"set_play_candidates"`
}

type RegisterEvidence struct {
	TriggerIndex int `json:"trigger_index"`
	EffectIndex  int `json:"effect_index"`
	Player       int `json:"player,omitempty"`
	Operation    int `json:"operation,omitempty"`
	Amount       int `json:"amount,omitempty"`
}

type RegisterMapResource struct {
	Resource   int                `json:"resource"`
	Role       string             `json:"role"`
	WriteCount int                `json:"write_count"`
	Amounts    []int              `json:"amounts,omitempty"`
	Players    []int              `json:"players,omitempty"`
	Evidence   []RegisterEvidence `json:"evidence,omitempty"`
}

type RegisterMapRow struct {
	Lane       int                   `json:"lane"`
	Label      string                `json:"label"`
	Players    []int                 `json:"players"`
	Metric     string                `json:"metric"`
	Layer      string                `json:"layer"`
	Resources  []RegisterMapResource `json:"resources"`
	Confidence string                `json:"confidence"`
	Notes      string                `json:"notes,omitempty"`
}

type Registry struct {
	Scenario string          `json:"scenario"`
	Summary  RegistrySummary `json:"summary"`
	Entries  []RegistryEntry `json:"entries"`
}
    Registry is the admission decision for a whole replay tree.

func BuildRegistry(paths []string, progress func(path string, id ScenarioIdentity)) *Registry
    BuildRegistry classifies replays and resolves duplicates.

    The same match reaches us from several player POVs and from both the
    local save archive and the crawled MS-API pulls, so admitting on filename
    alone would multiply every game. Two replays are the same match when their
    profile-id roster and their duration (bucketed to 10 s, absorbing POV
    recording jitter) agree.

func BuildRegistryWithWorkers(paths []string, workers int, progress func(path string, id ScenarioIdentity)) *Registry
    BuildRegistryWithWorkers classifies concurrently (workers <= 0 means
    NumCPU).

    Classification is parallel because each replay is independent, but
    DEDUPLICATION is a second, sequential pass in stable path order. Resolving
    duplicates inside the workers would make "which copy of this match got
    admitted" depend on scheduling, so the same corpus could produce different
    registries run to run.

func LoadRegistry(path string) (*Registry, error)
    LoadRegistry reads a registry written by WriteRegistry.

func (r *Registry) Admitted() []string
    Admitted returns the paths cleared for ingest.

type RegistryEntry struct {
	Path        string           `json:"path"`
	GameID      string           `json:"game_id"`
	Scenario    ScenarioIdentity `json:"scenario"`
	DurationS   int              `json:"duration_s,omitempty"`
	Players     []RegistryPlayer `json:"players,omitempty"`
	Source      string           `json:"source,omitempty"`
	Admit       bool             `json:"admit"`
	Reason      string           `json:"reason"`
	DuplicateOf string           `json:"duplicate_of,omitempty"`
}
    RegistryEntry is one classified replay.

type RegistryPlayer struct {
	Number    int    `json:"number"`
	Name      string `json:"name,omitempty"`
	ProfileID int    `json:"profile_id,omitempty"`
	Team      string `json:"team,omitempty"`
	Civ       int    `json:"civ,omitempty"`
}
    RegistryPlayer is one seat in a classified game.

type RegistrySummary struct {
	Scanned    int `json:"scanned"`
	Ladder     int `json:"ladder"`
	Admitted   int `json:"admitted"`
	Duplicates int `json:"duplicates"`
	Rejected   int `json:"rejected"`
	TooFew     int `json:"too_few_players"`
}

type ResearchFirstSeen struct {
	PlayerID     int    `json:"player_id"`
	PlayerLabel  string `json:"player_label,omitempty"`
	TimeMS       int    `json:"time_ms"`
	Time         string `json:"time"`
	TechnologyID int    `json:"technology_id"`
	Confidence   string `json:"confidence"`
}

type ScenarioIdentity struct {
	Token        string `json:"token,omitempty"`
	TriggerCount int    `json:"trigger_count,omitempty"`
	Ladder       bool   `json:"ladder"`
	Reason       string `json:"reason"`
}
    ScenarioIdentity is why a replay was admitted to, or rejected from,
    the ladder.

func IdentifyScenario(path string) ScenarioIdentity
    IdentifyScenario classifies one replay. The reason field is always populated
    so a rejection can be audited without re-running anything.

type ScoreboardChannelRow struct {
	Label          string `json:"label"`
	Players        []int  `json:"players"`
	KillResourceA  int    `json:"kill_resource_a"`
	KillResourceB  int    `json:"kill_resource_b"`
	DeathResourceA int    `json:"death_resource_a"`
	DeathResourceB int    `json:"death_resource_b"`
	RazeResourceA  int    `json:"raze_resource_a"`
	RazeResourceB  int    `json:"raze_resource_b"`
	TemplateLine   string `json:"template_line,omitempty"`
	Confidence     string `json:"confidence"`
}

type SetPlayCandidate struct {
	TimeMS           int                    `json:"time_ms"`
	Time             string                 `json:"time"`
	TargetID         int                    `json:"target_id"`
	TargetClass      string                 `json:"target_class,omitempty"`
	TargetOwnerID    int                    `json:"target_owner_id,omitempty"`
	TargetOwnerLabel string                 `json:"target_owner_label,omitempty"`
	WeakenerSequence []TargetPlayerPressure `json:"weakener_sequence,omitempty"`
	FinisherID       int                    `json:"finisher_id"`
	FinisherLabel    string                 `json:"finisher_label,omitempty"`
	VillagerObjectID int                    `json:"villager_object_id"`
	Confidence       string                 `json:"confidence"`
	Reason           string                 `json:"reason"`
}

type SideChannelNote struct {
	Name       string `json:"name"`
	Confidence string `json:"confidence"`
	Detail     string `json:"detail"`
}

type SideChannelReport struct {
	Path          string                 `json:"path,omitempty"`
	Method        string                 `json:"method"`
	Verification  string                 `json:"verification"`
	GraphSHA256   string                 `json:"graph_sha256,omitempty"`
	ScenarioToken string                 `json:"scenario_token,omitempty"`
	Scenario      ScenarioIdentity       `json:"scenario_identity"`
	DataSet       replay.DataSetIdentity `json:"data_set_identity"`
	Summary       SideChannelSummary     `json:"summary"`
	Scoreboard    []ScoreboardChannelRow `json:"scoreboard,omitempty"`
	EngineAttrs   []EngineAttributeRow   `json:"engine_attributes,omitempty"`
	Resources     []SideChannelResource  `json:"resources,omitempty"`
	RegisterMap   []RegisterMapRow       `json:"register_map,omitempty"`
	Thresholds    []SideChannelThreshold `json:"thresholds,omitempty"`
	Templates     []string               `json:"scoreboard_templates,omitempty"`
	Notes         []SideChannelNote      `json:"notes,omitempty"`
	Warnings      []string               `json:"warnings,omitempty"`
}

func BuildSideChannels(path string) (*SideChannelReport, error)

type SideChannelResource struct {
	Resource int    `json:"resource"`
	Family   string `json:"family"`
	Count    int    `json:"write_count"`
	Amounts  []int  `json:"amounts"`
	Players  []int  `json:"players"`
}

type SideChannelSummary struct {
	SideChannelVariant    string `json:"sidechannel_variant"`
	TriggerCount          int    `json:"trigger_count"`
	EffectCount           int    `json:"effect_count"`
	ConditionCount        int    `json:"condition_count"`
	MessageCount          int    `json:"message_count"`
	ModifyResourceEffects int    `json:"modify_resource_effects"`
	ChangeVariableEffects int    `json:"change_variable_effects"`
	AccumAttributeConds   int    `json:"accumulate_attribute_conditions"`
	ScoreboardRows        int    `json:"scoreboard_rows"`
	EngineAttrChannels    int    `json:"engine_attr_channels"`
	ResourcesWritten      int    `json:"resources_written"`
	ThresholdChannels     int    `json:"threshold_channels"`
}

type SideChannelThreshold struct {
	Kind      string   `json:"kind"`
	Attribute int      `json:"attribute"`
	Players   []int    `json:"players"`
	Values    []int    `json:"values"`
	Messages  []string `json:"messages,omitempty"`
}

type TargetPlayerPressure struct {
	PlayerID    int    `json:"player_id"`
	PlayerLabel string `json:"player_label,omitempty"`
	FirstTimeMS int    `json:"first_time_ms"`
	FirstTime   string `json:"first_time"`
	LastTimeMS  int    `json:"last_time_ms"`
	LastTime    string `json:"last_time"`
	Events      int    `json:"events"`
}

type TargetPressure struct {
	TargetID        int                    `json:"target_id"`
	OwnerID         int                    `json:"owner_id,omitempty"`
	OwnerLabel      string                 `json:"owner_label,omitempty"`
	Class           string                 `json:"class,omitempty"`
	UnitID          int                    `json:"unit_id,omitempty"`
	UnitName        string                 `json:"unit_name,omitempty"`
	X               float64                `json:"x,omitempty"`
	Y               float64                `json:"y,omitempty"`
	FirstTimeMS     int                    `json:"first_time_ms"`
	FirstTime       string                 `json:"first_time"`
	LastTimeMS      int                    `json:"last_time_ms"`
	LastTime        string                 `json:"last_time"`
	Events          int                    `json:"events"`
	DistinctPlayers int                    `json:"distinct_players"`
	Players         []TargetPlayerPressure `json:"players,omitempty"`
}

type TriggerRazeBucket struct {
	RazesToVillager int      `json:"razes_to_villager"`
	TechIDs         []int    `json:"technology_ids"`
	CivIDs          []int    `json:"civ_ids,omitempty"`
	CivNames        []string `json:"civ_names,omitempty"`
	Count           int      `json:"count"`
}

type TriggerRazeConflict struct {
	TechID int   `json:"technology_id"`
	Values []int `json:"values"`
}

type TriggerRazeMechanics struct {
	RazeAttribute       int    `json:"raze_attribute"`
	RazeAttributeName   string `json:"raze_attribute_name"`
	ConditionType       int    `json:"condition_type"`
	ConditionName       string `json:"condition_name"`
	RewardEffectType    int    `json:"reward_effect_type"`
	RewardEffectName    string `json:"reward_effect_name"`
	ActivatorEffectType int    `json:"activator_effect_type"`
	ActivatorEffectName string `json:"activator_effect_name"`
	Notes               string `json:"notes"`
}

type TriggerRazeReport struct {
	Path         string                `json:"path,omitempty"`
	DatPath      string                `json:"dat_path,omitempty"`
	Method       string                `json:"method"`
	Verification string                `json:"verification"`
	Summary      TriggerRazeSummary    `json:"summary"`
	Rows         []TriggerRazeRow      `json:"rows,omitempty"`
	Buckets      []TriggerRazeBucket   `json:"buckets,omitempty"`
	Mechanics    TriggerRazeMechanics  `json:"mechanics"`
	Conflicts    []TriggerRazeConflict `json:"conflicts,omitempty"`
	Warnings     []string              `json:"warnings,omitempty"`
}

func BuildTriggerRazes(path, datPath string) (*TriggerRazeReport, error)

type TriggerRazeRow struct {
	CivID           int    `json:"civ_id,omitempty"`
	CivName         string `json:"civ_name,omitempty"`
	TechID          int    `json:"technology_id"`
	TechName        string `json:"technology_name,omitempty"`
	RazesToVillager int    `json:"razes_to_villager"`
	TriggerID       int    `json:"trigger_id"`
	TargetTriggerID int    `json:"target_trigger_id"`
	SourcePlayers   []int  `json:"source_players,omitempty"`
	Message         string `json:"message,omitempty"`
	Confidence      string `json:"confidence"`
}

type TriggerRazeSummary struct {
	TriggerCount           int `json:"trigger_count"`
	RewardRungTriggers     int `json:"reward_rung_triggers"`
	ActivatorTriggers      int `json:"activator_triggers"`
	Rows                   int `json:"rows"`
	ResolvedCivs           int `json:"resolved_civs"`
	UniqueTechs            int `json:"unique_techs"`
	Buckets                int `json:"buckets"`
	Conflicts              int `json:"conflicts"`
	UnresolvedTechs        int `json:"unresolved_techs"`
	MapucheRazesToVillager int `json:"mapuche_razes_to_villager,omitempty"`
	KhmerRazesToVillager   int `json:"khmer_razes_to_villager,omitempty"`
}

type TriggerSpawnBucket struct {
	SpawnSeconds   *int     `json:"spawn_seconds,omitempty"`
	SpawnUnitCount *int     `json:"spawn_unit_count,omitempty"`
	TechIDs        []int    `json:"technology_ids"`
	CivIDs         []int    `json:"civ_ids,omitempty"`
	CivNames       []string `json:"civ_names,omitempty"`
	Count          int      `json:"count"`
}

type TriggerSpawnConflict struct {
	TechID int    `json:"technology_id"`
	Field  string `json:"field"`
	Values []int  `json:"values"`
}

type TriggerSpawnMechanics struct {
	SecondsSource       string `json:"seconds_source"`
	CountConditionType  int    `json:"count_condition_type"`
	CountConditionName  string `json:"count_condition_name"`
	SpawnEffectTypes    []int  `json:"spawn_effect_types"`
	SpawnEffectNames    string `json:"spawn_effect_names"`
	TechnologyCondition int    `json:"technology_condition"`
	Notes               string `json:"notes"`
}

type TriggerSpawnReport struct {
	Path         string                 `json:"path,omitempty"`
	DatPath      string                 `json:"dat_path,omitempty"`
	Method       string                 `json:"method"`
	Verification string                 `json:"verification"`
	Summary      TriggerSpawnSummary    `json:"summary"`
	Rows         []TriggerSpawnRow      `json:"rows,omitempty"`
	Buckets      []TriggerSpawnBucket   `json:"buckets,omitempty"`
	Mechanics    TriggerSpawnMechanics  `json:"mechanics"`
	Conflicts    []TriggerSpawnConflict `json:"conflicts,omitempty"`
	Warnings     []string               `json:"warnings,omitempty"`
}

func BuildTriggerSpawns(path, datPath string) (*TriggerSpawnReport, error)

type TriggerSpawnRow struct {
	CivID          int    `json:"civ_id,omitempty"`
	CivName        string `json:"civ_name,omitempty"`
	TechID         int    `json:"technology_id"`
	TechName       string `json:"technology_name,omitempty"`
	SpawnSeconds   *int   `json:"spawn_seconds,omitempty"`
	SpawnUnitCount *int   `json:"spawn_unit_count,omitempty"`
	SecondsTrigger int    `json:"seconds_trigger_id,omitempty"`
	CountTrigger   int    `json:"count_trigger_id,omitempty"`
	SourcePlayers  []int  `json:"source_players,omitempty"`
	Message        string `json:"message,omitempty"`
	Confidence     string `json:"confidence"`
}

type TriggerSpawnSummary struct {
	TriggerCount           int `json:"trigger_count"`
	SpawnerSecondTriggers  int `json:"spawner_second_triggers"`
	SpawnCountTriggers     int `json:"spawn_count_triggers"`
	Rows                   int `json:"rows"`
	CompleteRows           int `json:"complete_rows"`
	ResolvedCivs           int `json:"resolved_civs"`
	UniqueTechs            int `json:"unique_techs"`
	Buckets                int `json:"buckets"`
	Conflicts              int `json:"conflicts"`
	UnresolvedTechs        int `json:"unresolved_techs"`
	MapucheSpawnSeconds    int `json:"mapuche_spawn_seconds,omitempty"`
	MapucheSpawnUnitCount  int `json:"mapuche_spawn_unit_count,omitempty"`
	KhmerSpawnSeconds      int `json:"khmer_spawn_seconds,omitempty"`
	KhmerSpawnUnitCount    int `json:"khmer_spawn_unit_count,omitempty"`
	PersiansSpawnSeconds   int `json:"persians_spawn_seconds,omitempty"`
	PersiansSpawnUnitCount int `json:"persians_spawn_unit_count,omitempty"`
}
```

## aoe2kit/pkg/ci

```go
package ci // import "aoe2kit/pkg/ci"


CONSTANTS

const DefaultConfigName = "aoe2kit-ci.json"

TYPES

type Check struct {
	Name     string `json:"name,omitempty"`
	Type     string `json:"type"`
	Replay   string `json:"replay,omitempty"`
	Contract string `json:"contract,omitempty"`
	XSDat    string `json:"xsdat,omitempty"`
	Ledger   string `json:"ledger,omitempty"`
	Schema   string `json:"schema,omitempty"`
	Player   int    `json:"player,omitempty"`
	WindowMS int    `json:"window_ms,omitempty"`
}

type Config struct {
	SchemaVersion int     `json:"schema_version"`
	Name          string  `json:"name,omitempty"`
	Scenario      string  `json:"scenario,omitempty"`
	Checks        []Check `json:"checks"`
}

type InitReport struct {
	Root       string   `json:"root"`
	ConfigPath string   `json:"config_path"`
	DebugXS    string   `json:"debug_xs"`
	Created    []string `json:"created,omitempty"`
	Updated    []string `json:"updated,omitempty"`
}

func Init(root string, scenario string, overwrite bool) (*InitReport, error)

type Report struct {
	Path    string   `json:"path"`
	Root    string   `json:"root"`
	Name    string   `json:"name,omitempty"`
	OK      bool     `json:"ok"`
	Summary Summary  `json:"summary"`
	Checks  []Result `json:"checks"`
	Warning []string `json:"warnings,omitempty"`
	Config  *Config  `json:"config,omitempty"`
}

func CheckFile(path string) (*Report, error)

type Result struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Source   string `json:"source,omitempty"`
	Message  string `json:"message,omitempty"`
	Replay   string `json:"replay,omitempty"`
	Contract string `json:"contract,omitempty"`
	XSDat    string `json:"xsdat,omitempty"`
	Ledger   string `json:"ledger,omitempty"`
	Schema   string `json:"schema,omitempty"`
	Details  any    `json:"details,omitempty"`
}

type Summary struct {
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Unknown int `json:"unknown"`
}
```

## aoe2kit/pkg/customstore

```go
package customstore // import "aoe2kit/pkg/customstore"

Package customstore provides a generic, scale-tested per-instance store recipe.
It deliberately contains no scenario-specific item effects.

CONSTANTS

const (
	DefaultObjectUnitConst        = 236 // WCTW4 / Tower WEST Bombard, demo only
	DefaultPlayerCount            = 8
	DefaultInstances              = 440
	InstancesPerPlayer            = DefaultInstances / DefaultPlayerCount
	MapWidth                      = 120
	MapHeight                     = 120
	ReferenceIDBase               = 920000
	KeepAliveUnitConst            = 12 // Barracks; prevents DE from injecting a TC
	DefaultStartingResourceAmount = 100000
	FireUnitConst                 = 939  // FLAME1; HP-retired example purchase effect
	FlowerUnitConst               = 1366 // footprintless class-14 Gaia art; kill-zone retired
	FireCapacity                  = 96
	FireLifetimeSeconds           = 12
	KillSweepSeconds              = 120
	TokenUpgradeUnit              = 74 // placeholder token; content is intentionally not defined here
	TokenInspectUnit              = 93 // placeholder token; content is intentionally not defined here
	// The blank scenario retains one harmless seed trigger; the store adds setup,
	// polling, telemetry, and one bounded Gaia kill-zone sweep per second.
	TriggerCount        = 1 + AddedTriggerCount
	AddedTriggerCount   = 3 + KillSweepSeconds
	TriggerVariableHigh = 6
)

FUNCTIONS

func Recipe(xsPath string, timestamp int, objectUnitConst, instanceCount, playerCount int) scenario.Recipe
func RecipeForPlayers(xsPath string, timestamp int, objectUnitConst, instanceCount, playerCount int, activePlayers []int) scenario.Recipe
    RecipeForPlayers keeps the full object/player scale while isolating store
    interaction to the supplied participants. Non-participants retain their
    objects for scale and their normal scenario slots, but receive no store UI.

func SidecarTypes() string
    SidecarTypes documents the exact xsWrite order emitted by XSModule.
    It is intentionally plain text so external scenario tooling can consume the
    fixture without importing AoE2Kit.

func XSModule(objectUnitConst, instanceCount, playerCount int) string
func XSModuleForPlayers(objectUnitConst, instanceCount, playerCount int, activePlayers []int) string
    XSModuleForPlayers emits the same bounded runtime for the full object scale,
    but consumes tokens only for the selected store participants.


TYPES

type Options struct {
	OutputDir       string
	ScenarioName    string
	Timestamp       int
	ObjectUnitConst int
	InstanceCount   int
	PlayerCount     int
	// ActivePlayers identifies the players allowed to interact with the store.
	// Empty means every configured player, preserving the scale-test behavior.
	ActivePlayers  []int
	XSCheckPath    string
	RequireXSCheck bool
}

type Report struct {
	OutputDir          string              `json:"output_dir"`
	ScenarioPath       string              `json:"scenario_path"`
	BaseScenarioPath   string              `json:"base_scenario_path"`
	RecipePath         string              `json:"recipe_path"`
	XSPath             string              `json:"xs_path"`
	TypesPath          string              `json:"types_path"`
	ScenarioSHA256     string              `json:"scenario_sha256"`
	XSSHA256           string              `json:"xs_sha256"`
	ObjectUnitConst    int                 `json:"object_unit_const"`
	InstanceCount      int                 `json:"instance_count"`
	InstancesPerPlayer int                 `json:"instances_per_player"`
	PlayerCount        int                 `json:"player_count"`
	ActivePlayers      []int               `json:"active_players"`
	TriggerCount       int                 `json:"trigger_count"`
	UnitCount          int                 `json:"unit_count"`
	Verification       string              `json:"verification"`
	KnownGaps          []string            `json:"known_gaps"`
	XSParseCheck       string              `json:"xs_parse_check"`
	XSParseChecker     string              `json:"xs_parse_checker,omitempty"`
	SemanticLint       scenario.LintReport `json:"semantic_lint"`
}

func Build(opts Options) (*Report, error)
```

## aoe2kit/pkg/datcache

```go
package datcache // import "aoe2kit/pkg/datcache"


TYPES

type Cache struct {
	Path     string
	SHA256   string
	Version  string
	Built    bool
	CivCount int
	// Has unexported fields.
}

func Open(datPath string) (*Cache, error)

func (c *Cache) Close() error

func (c *Cache) QueryUnits(query UnitQuery) ([]UnitRow, int, error)

func (c *Cache) SemanticPriors() (SemanticPriorsReport, error)
    SemanticPriors derives the first-pass prediction tables from the cached DAT.
    Unit identities are grouped by unit_index so civ duplication does not skew
    the result. A max aggregation mirrors the canonical SQL used in the ledger.

func (c *Cache) Unit(civID, unitID int) (datfile.UnitSummary, bool, error)

func (c *Cache) Units() ([]UnitRow, error)

type CollisionSignature struct {
	XPositive bool `json:"x_positive"`
	YPositive bool `json:"y_positive"`
	ZPositive bool `json:"z_positive"`
	Units     int  `json:"units"`
	Creatable int  `json:"creatable"`
}

type RemovalBucket struct {
	Name  string `json:"name"`
	Units int    `json:"units"`
}

type SemanticPriorsReport struct {
	Version                string               `json:"version"`
	CivCount               int                  `json:"civ_count"`
	SourceRows             int                  `json:"source_rows"`
	UnitIdentities         int                  `json:"unit_identities"`
	CollisionSignatures    []CollisionSignature `json:"collision_signatures"`
	RemovalBuckets         []RemovalBucket      `json:"removal_buckets"`
	ZeroCollision          int                  `json:"zero_collision_identities"`
	ZeroCollisionCreatable int                  `json:"zero_collision_creatable"`
	Verification           string               `json:"verification"`
}
    SemanticPriorsReport summarizes DAT-derived hypotheses. It deliberately
    does not claim engine semantics; callers should compare it with a harness
    observation before upgrading any row to an engine fact.

type UnitQuery struct {
	Limit        int
	All          bool
	Civ          *int
	ID           *int
	Name         string
	NameContains string
	Class        string
}

type UnitRow struct {
	CivIndex           int
	Index              int
	Type               int
	ID                 int16
	Name               string
	Class              int16
	ClassName          string
	HitPoints          int16
	LineOfSight        float32
	CollisionSizeX     float32
	CollisionSizeY     float32
	CollisionSizeZ     float32
	ClearanceSizeX     float32
	ClearanceSizeY     float32
	TerrainTableID     int16
	MovementType       uint8
	ObstructionTypeID  uint8
	ObstructionClassID uint8
	IconID             int16
	Enabled            uint8
	HasType50          bool
	HasCreatable       bool
	RecordStart        int
	RecordEnd          int
}
```

## aoe2kit/pkg/datcodec

```go
package datcodec // import "aoe2kit/pkg/datcodec"


CONSTANTS

const Version = "aoe2kit.datcodec.v1"

FUNCTIONS

func PackEffectAttackArmorAmount(classID int, amount int) (float32, error)
func RecipeAuthoringWarnings(recipe Recipe) []string

TYPES

type AbilityCreateRecipe struct {
	Name                   string                         `json:"name"`
	EffectName             string                         `json:"effect_name,omitempty"`
	FromTech               int                            `json:"from_tech"`
	FromEffect             *int                           `json:"from_effect,omitempty"`
	Commands               []EffectCommandRecipe          `json:"commands,omitempty"`
	AppendCommands         []EffectCommandRecipe          `json:"append_commands,omitempty"`
	RequiredTechs          []int16                        `json:"required_techs,omitempty"`
	ResourceCosts          []datfile.ResearchResourceCost `json:"resource_costs,omitempty"`
	RequiredTechCount      *int16                         `json:"required_tech_count,omitempty"`
	Civ                    *int16                         `json:"civ,omitempty"`
	FullTechMode           *int16                         `json:"full_tech_mode,omitempty"`
	LanguageDLLName        *int32                         `json:"language_dll_name,omitempty"`
	LanguageDLLDescription *int32                         `json:"language_dll_description,omitempty"`
	Type                   *int16                         `json:"type,omitempty"`
	IconID                 *int16                         `json:"icon_id,omitempty"`
	LanguageDLLHelp        *int32                         `json:"language_dll_help,omitempty"`
	LanguageDLLTechTree    *int32                         `json:"language_dll_tech_tree,omitempty"`
	Repeatable             *uint8                         `json:"repeatable,omitempty"`
	ResearchLocations      []datfile.ResearchLocation     `json:"research_locations,omitempty"`
}

type AbilityCreateReport struct {
	Name     string `json:"name"`
	TechID   int    `json:"tech_id"`
	EffectID int    `json:"effect_id"`
	Strategy string `json:"strategy"`
}

type AbilityDeleteReport struct {
	TechID   int    `json:"tech_id"`
	EffectID int    `json:"effect_id"`
	Strategy string `json:"strategy"`
	Note     string `json:"note,omitempty"`
}

type CivPatchRecipe struct {
	ID          int                `json:"id"`
	Name        *string            `json:"name,omitempty"`
	TechTreeID  *int16             `json:"tech_tree_id,omitempty"`
	TeamBonusID *int16             `json:"team_bonus_id,omitempty"`
	IconSet     *uint8             `json:"icon_set,omitempty"`
	Resources   []CivResourcePatch `json:"resources,omitempty"`
}

type CivResourcePatch struct {
	Index int     `json:"index"`
	Value float32 `json:"value"`
}

type CivTarget struct {
	Count        int               `json:"count"`
	Resources    int               `json:"resources"`
	PresentUnits int               `json:"present_units"`
	Spans        []TypedRecordSpan `json:"spans,omitempty"`
	Records      []datfile.Civ     `json:"records,omitempty"`
	Editable     []string          `json:"editable"`
}

type ClassifiedReference struct {
	DeleteReference
	Class  string `json:"class"`
	Reason string `json:"reason,omitempty"`
}

type DeletePlanReport struct {
	Path           string                     `json:"path,omitempty"`
	Version        string                     `json:"version"`
	Request        DeletePlanRequest          `json:"request"`
	Supported      bool                       `json:"supported"`
	Mutates        bool                       `json:"mutates"`
	Strategy       string                     `json:"strategy"`
	RecipeHint     json.RawMessage            `json:"recipe_hint,omitempty"`
	CleanupCommand string                     `json:"cleanup_command,omitempty"`
	CleanupRecipe  json.RawMessage            `json:"cleanup_recipe,omitempty"`
	References     []DeleteReference          `json:"references,omitempty"`
	TechMigration  *TechDeleteMigrationReport `json:"tech_migration,omitempty"`
	Warnings       []string                   `json:"warnings,omitempty"`
	Reason         string                     `json:"reason,omitempty"`
	Verification   aoe2.VerificationClaim     `json:"verification"`
}

func DeletePlan(idx *datfile.Index, request DeletePlanRequest) DeletePlanReport

func DeletePlanFile(path string, request DeletePlanRequest) (DeletePlanReport, error)

type DeletePlanRequest struct {
	Section      string `json:"section"`
	ID           int    `json:"id"`
	EffectID     *int   `json:"effect_id,omitempty"`
	CommandIndex *int   `json:"command_index,omitempty"`
	SoundID      *int   `json:"sound_id,omitempty"`
	ItemIndex    *int   `json:"item_index,omitempty"`
	GraphicID    *int   `json:"graphic_id,omitempty"`
	UnitHeaderID *int   `json:"unit_header_id,omitempty"`
	UnitID       *int   `json:"unit_id,omitempty"`
	RowIndex     *int   `json:"row_index,omitempty"`
	CivID        *int   `json:"civ_id,omitempty"`
	AllCivs      bool   `json:"all_civs,omitempty"`
}

type DeleteReference struct {
	Section    string `json:"section"`
	ID         int    `json:"id"`
	Field      string `json:"field"`
	Confidence string `json:"confidence,omitempty"`
}

type Document struct {
	Version       string                 `json:"version"`
	DatVersion    string                 `json:"dat_version"`
	InflatedBytes int                    `json:"inflated_bytes"`
	Sections      []Section              `json:"sections,omitempty"`
	Targets       TypedTargets           `json:"typed_targets"`
	Verification  aoe2.VerificationClaim `json:"verification"`
	// Has unexported fields.
}

func Decode(compressed []byte) (*Document, error)

func (doc *Document) EncodePayload() ([]byte, error)

type EffectCommandExample struct {
	EffectID   int                              `json:"effect_id"`
	EffectName string                           `json:"effect_name"`
	Command    datfile.EffectCommand            `json:"command"`
	References []datfile.EffectCommandReference `json:"references,omitempty"`
}

type EffectCommandExplanation struct {
	Index      int                              `json:"index"`
	Type       uint8                            `json:"type"`
	TypeName   string                           `json:"type_name"`
	Scope      string                           `json:"scope,omitempty"`
	Raw        datfile.EffectCommand            `json:"raw"`
	Summary    string                           `json:"summary"`
	Details    []string                         `json:"details,omitempty"`
	Warnings   []string                         `json:"warnings,omitempty"`
	References []datfile.EffectCommandReference `json:"references,omitempty"`
	Recipe     EffectCommandRecipe              `json:"recipe"`
	Confidence string                           `json:"confidence,omitempty"`
	Source     string                           `json:"source,omitempty"`
}

type EffectCommandMatrixFilters struct {
	CommandType    *int   `json:"command_type,omitempty"`
	Operand        *int   `json:"operand,omitempty"`
	ReferenceKind  string `json:"reference_kind,omitempty"`
	ReferenceField string `json:"reference_field,omitempty"`
	UnknownOnly    bool   `json:"unknown_only,omitempty"`
	TypedOnly      bool   `json:"typed_only,omitempty"`
	CandidateOnly  bool   `json:"candidate_only,omitempty"`
}

type EffectCommandMatrixOptions struct {
	ExampleLimit   int
	CommandType    *int
	Operand        *int
	ReferenceKind  string
	ReferenceField string
	UnknownOnly    bool
	TypedOnly      bool
	CandidateOnly  bool
}

type EffectCommandMatrixReport struct {
	Version             string                     `json:"version"`
	Method              string                     `json:"method"`
	Verification        aoe2.VerificationClaim     `json:"verification"`
	Filters             EffectCommandMatrixFilters `json:"filters,omitempty"`
	EffectCount         int                        `json:"effect_count"`
	MatchingEffectCount int                        `json:"matching_effect_count,omitempty"`
	CommandCount        int                        `json:"command_count"`
	TypeCount           int                        `json:"type_count"`
	ExampleLimit        int                        `json:"example_limit"`
	CommandTypes        []EffectCommandTypeProfile `json:"command_types"`
}

func EffectCommandMatrix(idx *datfile.Index, exampleLimit int) EffectCommandMatrixReport

func EffectCommandMatrixWithOptions(idx *datfile.Index, opts EffectCommandMatrixOptions) EffectCommandMatrixReport

type EffectCommandOperandProfile struct {
	Min             float64                   `json:"min"`
	Max             float64                   `json:"max"`
	Negative        int                       `json:"negative"`
	Zero            int                       `json:"zero"`
	Positive        int                       `json:"positive"`
	Integral        int                       `json:"integral"`
	Distinct        int                       `json:"distinct"`
	SampleValues    []EffectCommandValueCount `json:"sample_values,omitempty"`
	CommonValues    []EffectCommandValueCount `json:"common_values,omitempty"`
	LooksLikeIDList bool                      `json:"looks_like_id_list,omitempty"`
}

type EffectCommandOperandProfiles struct {
	A EffectCommandOperandProfile `json:"a"`
	B EffectCommandOperandProfile `json:"b"`
	C EffectCommandOperandProfile `json:"c"`
	D EffectCommandOperandProfile `json:"d"`
}

type EffectCommandRecipe struct {
	Type              uint8    `json:"type"`
	A                 int16    `json:"a"`
	B                 int16    `json:"b"`
	C                 int16    `json:"c"`
	D                 float32  `json:"d"`
	Kind              string   `json:"kind,omitempty"`
	UnitID            *int16   `json:"unit_id,omitempty"`
	ToUnitID          *int16   `json:"to_unit_id,omitempty"`
	BuildingID        *int16   `json:"building_id,omitempty"`
	UnitClassID       *int16   `json:"unit_class_id,omitempty"`
	UnitClass         string   `json:"unit_class,omitempty"`
	UnitClassName     string   `json:"unit_class_name,omitempty"`
	AttributeID       *int16   `json:"attribute_id,omitempty"`
	Attribute         string   `json:"attribute,omitempty"`
	AttributeName     string   `json:"attribute_name,omitempty"`
	PackedTypeID      *int     `json:"packed_type_id,omitempty"`
	PackedType        string   `json:"packed_type,omitempty"`
	PackedTypeName    string   `json:"packed_type_name,omitempty"`
	PackedAmount      *int     `json:"packed_amount,omitempty"`
	ArmorClassID      *int     `json:"armor_class_id,omitempty"`
	ArmorClass        string   `json:"armor_class,omitempty"`
	AttackClassID     *int     `json:"attack_class_id,omitempty"`
	AttackClass       string   `json:"attack_class,omitempty"`
	CombatClassID     *int     `json:"combat_class_id,omitempty"`
	CombatClass       string   `json:"combat_class,omitempty"`
	ResourceID        *int16   `json:"resource_id,omitempty"`
	Resource          string   `json:"resource,omitempty"`
	ResourceName      string   `json:"resource_name,omitempty"`
	OperationID       *int16   `json:"operation_id,omitempty"`
	Operation         string   `json:"operation,omitempty"`
	OperationName     string   `json:"operation_name,omitempty"`
	TechAttrID        *int16   `json:"tech_attribute_id,omitempty"`
	TechAttribute     string   `json:"tech_attribute,omitempty"`
	TechAttributeName string   `json:"tech_attribute_name,omitempty"`
	Amount            *float32 `json:"amount,omitempty"`
	TechID            *int     `json:"tech_id,omitempty"`
}

type EffectCommandReferenceCount struct {
	Kind      string                    `json:"kind"`
	Field     string                    `json:"field"`
	Count     int                       `json:"count"`
	SampleIDs []EffectCommandValueCount `json:"sample_ids,omitempty"`
}

type EffectCommandTypeProfile struct {
	Type             uint8                         `json:"type"`
	TypeName         string                        `json:"type_name"`
	CommandCount     int                           `json:"command_count"`
	EffectCount      int                           `json:"effect_count"`
	Operands         EffectCommandOperandProfiles  `json:"operands"`
	TypedReferences  []EffectCommandReferenceCount `json:"typed_references,omitempty"`
	CandidateRefs    []EffectCommandReferenceCount `json:"candidate_references,omitempty"`
	Attributes       []EffectCommandValueCount     `json:"attributes,omitempty"`
	PackedTypeIDs    []EffectCommandValueCount     `json:"packed_type_ids,omitempty"`
	PackedAmounts    []EffectCommandValueCount     `json:"packed_amounts,omitempty"`
	ResourceIDs      []EffectCommandValueCount     `json:"resource_ids,omitempty"`
	OperationIDs     []EffectCommandValueCount     `json:"operation_ids,omitempty"`
	TechAttributeIDs []EffectCommandValueCount     `json:"tech_attribute_ids,omitempty"`
	Examples         []EffectCommandExample        `json:"examples,omitempty"`
}

type EffectCommandValueCount struct {
	Value int    `json:"value"`
	Count int    `json:"count"`
	Name  string `json:"name,omitempty"`
}

type EffectCreateRecipe struct {
	Name           string                `json:"name"`
	FromEffect     *int                  `json:"from_effect,omitempty"`
	Commands       []EffectCommandRecipe `json:"commands,omitempty"`
	RemoveCommands []int                 `json:"remove_commands,omitempty"`
	AppendCommands []EffectCommandRecipe `json:"append_commands,omitempty"`
}

type EffectExplainReport struct {
	Version      string                     `json:"version"`
	EffectID     int                        `json:"effect_id"`
	EffectName   string                     `json:"effect_name"`
	Summary      string                     `json:"summary"`
	CommandCount int                        `json:"command_count"`
	Commands     []EffectCommandExplanation `json:"commands"`
	Warnings     []string                   `json:"warnings,omitempty"`
	Effect       datfile.Effect             `json:"effect"`
	Verification aoe2.VerificationClaim     `json:"verification"`
}

func ExplainEffect(idx *datfile.Index, effectID int) (EffectExplainReport, error)

type EffectPatchRecipe struct {
	ID             int                    `json:"id"`
	Name           *string                `json:"name,omitempty"`
	Commands       *[]EffectCommandRecipe `json:"commands,omitempty"`
	RemoveCommands []int                  `json:"remove_commands,omitempty"`
	AppendCommands []EffectCommandRecipe  `json:"append_commands,omitempty"`
}

type EffectTarget struct {
	Count    int               `json:"count"`
	Commands int               `json:"commands"`
	Spans    []TypedRecordSpan `json:"spans,omitempty"`
	Records  []datfile.Effect  `json:"records,omitempty"`
	Editable []string          `json:"editable"`
}

type GraphicTarget struct {
	Count         int      `json:"count"`
	Present       int      `json:"present"`
	Deltas        int      `json:"deltas"`
	AngleSounds   int      `json:"angle_sounds"`
	ParticleBound int      `json:"particle_bound"`
	Editable      []string `json:"editable"`
	DecodedFields []string `json:"decoded_fields"`
}

type PatchReport struct {
	Version                            string                        `json:"version"`
	InputCompressedBytes               int                           `json:"input_compressed_bytes"`
	OutputCompressedBytes              int                           `json:"output_compressed_bytes"`
	InputInflatedBytes                 int                           `json:"input_inflated_bytes"`
	OutputInflatedBytes                int                           `json:"output_inflated_bytes"`
	InflatedLengthDelta                int                           `json:"inflated_length_delta"`
	BeforeEffectCount                  int                           `json:"before_effect_count"`
	AfterEffectCount                   int                           `json:"after_effect_count"`
	BeforeGraphicCount                 int                           `json:"before_graphic_count,omitempty"`
	AfterGraphicCount                  int                           `json:"after_graphic_count,omitempty"`
	BeforeCivCount                     int                           `json:"before_civ_count"`
	AfterCivCount                      int                           `json:"after_civ_count"`
	BeforeTechCount                    int                           `json:"before_tech_count"`
	AfterTechCount                     int                           `json:"after_tech_count"`
	BeforeSoundCount                   int                           `json:"before_sound_count"`
	AfterSoundCount                    int                           `json:"after_sound_count"`
	BeforePlayerColourCount            int                           `json:"before_player_colour_count,omitempty"`
	AfterPlayerColourCount             int                           `json:"after_player_colour_count,omitempty"`
	CreatedAbilities                   []AbilityCreateReport         `json:"created_abilities,omitempty"`
	DisabledAbilities                  []AbilityDeleteReport         `json:"disabled_abilities,omitempty"`
	DeletedAbilities                   []AbilityDeleteReport         `json:"deleted_abilities,omitempty"`
	CreatedEffects                     []int                         `json:"created_effects,omitempty"`
	DisabledEffects                    []int                         `json:"disabled_effects,omitempty"`
	GraphicReports                     []datfile.PatchReport         `json:"graphic_reports,omitempty"`
	CreatedGraphics                    []datfile.GraphicCreateReport `json:"created_graphics,omitempty"`
	DeletedGraphics                    []datfile.GraphicDeleteReport `json:"deleted_graphics,omitempty"`
	CreatedUnits                       []datfile.UnitCreateReport    `json:"created_units,omitempty"`
	UnitReports                        []datfile.UnitPatchReport     `json:"unit_reports,omitempty"`
	CreatedTechs                       []int                         `json:"created_techs,omitempty"`
	CreatedPlayerColours               []int                         `json:"created_player_colours,omitempty"`
	DeletedPlayerColours               []PlayerColourDeleteReport    `json:"deleted_player_colours,omitempty"`
	CreatedSounds                      []int                         `json:"created_sounds,omitempty"`
	DeletedEffects                     []int                         `json:"deleted_effects,omitempty"`
	DeletedTechs                       []int                         `json:"deleted_techs,omitempty"`
	DeletedTechDetails                 []TechDeleteReport            `json:"deleted_tech_details,omitempty"`
	DeletedSounds                      []SoundDeleteReport           `json:"deleted_sounds,omitempty"`
	DeletedUnits                       []UnitDeleteReport            `json:"deleted_units,omitempty"`
	UnitAvailability                   []UnitAvailabilityReport      `json:"unit_availability,omitempty"`
	PatchedEffects                     []int                         `json:"patched_effects,omitempty"`
	PatchedUnitHeaders                 []int                         `json:"patched_unit_headers,omitempty"`
	PatchedCivs                        []int                         `json:"patched_civs,omitempty"`
	PatchedTerrainRestrictions         []int                         `json:"patched_terrain_restrictions,omitempty"`
	PatchedTerrains                    []int                         `json:"patched_terrains,omitempty"`
	PatchedSounds                      []int                         `json:"patched_sounds,omitempty"`
	PatchedPlayerColours               []int                         `json:"patched_player_colours,omitempty"`
	PatchedTechs                       []int                         `json:"patched_techs,omitempty"`
	DisconnectedTechs                  []TechDisconnectReport        `json:"disconnected_techs,omitempty"`
	DisconnectedUnits                  []UnitDisconnectReport        `json:"disconnected_units,omitempty"`
	PatchedTechTree                    bool                          `json:"patched_tech_tree,omitempty"`
	CreatedTechTreeBuildingConnections []int                         `json:"created_tech_tree_building_connections,omitempty"`
	CreatedTechTreeUnitConnections     []int                         `json:"created_tech_tree_unit_connections,omitempty"`
	CreatedTechTreeResearchConnections []int                         `json:"created_tech_tree_research_connections,omitempty"`
	DeletedTechTreeBuildingConnections []int                         `json:"deleted_tech_tree_building_connections,omitempty"`
	DeletedTechTreeUnitConnections     []int                         `json:"deleted_tech_tree_unit_connections,omitempty"`
	DeletedTechTreeResearchConnections []int                         `json:"deleted_tech_tree_research_connections,omitempty"`
	TechTreeRewrites                   []TechTreeRewriteReport       `json:"tech_tree_rewrites,omitempty"`
	ReferenceRewrites                  []ReferenceRewriteReport      `json:"reference_rewrites,omitempty"`
	AuthoringNotes                     []string                      `json:"authoring_notes,omitempty"`
	PayloadRoundTripOK                 bool                          `json:"payload_roundtrip_ok"`
	ReadbackOK                         bool                          `json:"readback_ok"`
	Verified                           bool                          `json:"verified"`
	Verification                       aoe2.VerificationClaim        `json:"verification"`
}

func PatchRecipe(compressed []byte, recipe Recipe) ([]byte, PatchReport, error)

func PatchRecipeFile(inputPath, outputPath string, recipe Recipe) (PatchReport, error)

func PlanRecipeFile(inputPath string, recipe Recipe) (PatchReport, error)

type PlayerColourCreateRecipe struct {
	From                int    `json:"from"`
	ColourID            *int32 `json:"colour_id,omitempty"`
	Base                *int32 `json:"base,omitempty"`
	UnitOutlineColour   *int32 `json:"unit_outline_colour,omitempty"`
	SelectionColour1    *int32 `json:"selection_colour_1,omitempty"`
	SelectionColour2    *int32 `json:"selection_colour_2,omitempty"`
	MinimapColour1      *int32 `json:"minimap_colour_1,omitempty"`
	MinimapColour2      *int32 `json:"minimap_colour_2,omitempty"`
	MinimapColour3      *int32 `json:"minimap_colour_3,omitempty"`
	StatisticsTextColor *int32 `json:"statistics_text_color,omitempty"`
}

type PlayerColourDeleteReport struct {
	ID          int    `json:"id"`
	BeforeCount int    `json:"before_count"`
	AfterCount  int    `json:"after_count"`
	Strategy    string `json:"strategy"`
	Note        string `json:"note,omitempty"`
}

type PlayerColourPatchRecipe struct {
	ID                  int    `json:"id"`
	ColourID            *int32 `json:"colour_id,omitempty"`
	Base                *int32 `json:"base,omitempty"`
	UnitOutlineColour   *int32 `json:"unit_outline_colour,omitempty"`
	SelectionColour1    *int32 `json:"selection_colour_1,omitempty"`
	SelectionColour2    *int32 `json:"selection_colour_2,omitempty"`
	MinimapColour1      *int32 `json:"minimap_colour_1,omitempty"`
	MinimapColour2      *int32 `json:"minimap_colour_2,omitempty"`
	MinimapColour3      *int32 `json:"minimap_colour_3,omitempty"`
	StatisticsTextColor *int32 `json:"statistics_text_color,omitempty"`
}

type PlayerColourTarget struct {
	Count    int      `json:"count"`
	Editable []string `json:"editable"`
}

type RandomMapTarget struct {
	Count      int    `json:"count"`
	Lands      int    `json:"lands"`
	Terrains   int    `json:"terrains"`
	Units      int    `json:"units"`
	Elevations int    `json:"elevations"`
	Note       string `json:"note,omitempty"`
}

type Recipe struct {
	CreateAbility       *AbilityCreateRecipe            `json:"create_ability,omitempty"`
	CreateAbilities     []AbilityCreateRecipe           `json:"create_abilities,omitempty"`
	DisableAbility      *int                            `json:"disable_ability,omitempty"`
	DisableAbilities    []int                           `json:"disable_abilities,omitempty"`
	DeleteAbility       *int                            `json:"delete_ability,omitempty"`
	DeleteAbilities     []int                           `json:"delete_abilities,omitempty"`
	CreateEffect        *EffectCreateRecipe             `json:"create_effect,omitempty"`
	CreateEffects       []EffectCreateRecipe            `json:"create_effects,omitempty"`
	DisableEffect       *int                            `json:"disable_effect,omitempty"`
	DisableEffects      []int                           `json:"disable_effects,omitempty"`
	Effects             []EffectPatchRecipe             `json:"effects,omitempty"`
	DeleteEffects       []int                           `json:"delete_effects,omitempty"`
	CreateGraphic       *datfile.GraphicCreateRecipe    `json:"create_graphic,omitempty"`
	CreateGraphics      []datfile.GraphicCreateRecipe   `json:"create_graphics,omitempty"`
	DeleteGraphics      []int                           `json:"delete_graphics,omitempty"`
	Graphics            []datfile.GraphicRecipePatch    `json:"graphics,omitempty"`
	UnitHeaders         []UnitHeaderPatchRecipe         `json:"unit_headers,omitempty"`
	CreateUnit          *datfile.UnitCreateRecipe       `json:"create_unit,omitempty"`
	CreateUnits         []datfile.UnitCreateRecipe      `json:"create_units,omitempty"`
	Units               []datfile.UnitRecipePatch       `json:"units,omitempty"`
	Civs                []CivPatchRecipe                `json:"civs,omitempty"`
	TerrainRestrictions []TerrainRestrictionPatchRecipe `json:"terrain_restrictions,omitempty"`
	Terrains            []TerrainPatchRecipe            `json:"terrains,omitempty"`
	CreateSound         *SoundCreateRecipe              `json:"create_sound,omitempty"`
	CreateSounds        []SoundCreateRecipe             `json:"create_sounds,omitempty"`
	Sounds              []SoundPatchRecipe              `json:"sounds,omitempty"`
	DeleteSounds        []int                           `json:"delete_sounds,omitempty"`
	DeleteUnits         []UnitDeleteRecipe              `json:"delete_units,omitempty"`
	UnitAvailability    []UnitAvailabilityRecipe        `json:"unit_availability,omitempty"`
	CreatePlayerColour  *PlayerColourCreateRecipe       `json:"create_player_colour,omitempty"`
	CreatePlayerColours []PlayerColourCreateRecipe      `json:"create_player_colours,omitempty"`
	PlayerColours       []PlayerColourPatchRecipe       `json:"player_colours,omitempty"`
	DeletePlayerColour  *int                            `json:"delete_player_colour,omitempty"`
	DeletePlayerColours []int                           `json:"delete_player_colours,omitempty"`
	CreateTech          *TechCreateRecipe               `json:"create_tech,omitempty"`
	CreateTechs         []TechCreateRecipe              `json:"create_techs,omitempty"`
	Techs               []TechPatchRecipe               `json:"techs,omitempty"`
	DeleteTechs         []int                           `json:"delete_techs,omitempty"`
	DisconnectTechs     []int                           `json:"disconnect_techs,omitempty"`
	DisconnectUnits     []int                           `json:"disconnect_units,omitempty"`
	TechTree            *TechTreePatchRecipe            `json:"tech_tree,omitempty"`
	ReferenceRewrites   []TechTreeIDRewriteRecipe       `json:"reference_rewrites,omitempty"`
}

func (recipe Recipe) Empty() bool

type ReferenceFilters struct {
	Class         string `json:"class,omitempty"`
	Confidence    string `json:"confidence,omitempty"`
	SourceSection string `json:"source_section,omitempty"`
	Limit         int    `json:"limit,omitempty"`
}

type ReferenceReport struct {
	Version      string                 `json:"version"`
	Request      DeletePlanRequest      `json:"request"`
	Filters      ReferenceFilters       `json:"filters,omitempty"`
	References   []ClassifiedReference  `json:"references,omitempty"`
	Summary      ReferenceSummary       `json:"summary"`
	Verification aoe2.VerificationClaim `json:"verification"`
}

func FilterReferenceReport(report ReferenceReport, filters ReferenceFilters) ReferenceReport

func References(idx *datfile.Index, request DeletePlanRequest) ReferenceReport

func ReferencesFile(path string, request DeletePlanRequest) (ReferenceReport, error)

type ReferenceRewriteReport struct {
	Target                  string                 `json:"target"`
	From                    int32                  `json:"from"`
	To                      *int32                 `json:"to,omitempty"`
	Remove                  bool                   `json:"remove"`
	TechRequiredRefs        int                    `json:"tech_required_refs"`
	EffectCommandsRewritten int                    `json:"effect_commands_rewritten"`
	EffectCommandsRemoved   int                    `json:"effect_commands_removed"`
	TechTree                *TechTreeRewriteReport `json:"tech_tree,omitempty"`
}

type ReferenceSummary struct {
	Total            int `json:"total"`
	Returned         int `json:"returned"`
	RewriteSupported int `json:"rewrite_supported"`
	KnownReadonly    int `json:"known_readonly"`
	PossibleOperand  int `json:"possible_operand"`
	CandidateOperand int `json:"candidate_operand"`
	Unsupported      int `json:"unsupported"`
}

type RoundTripReport struct {
	Version               string                 `json:"version"`
	Path                  string                 `json:"path"`
	OK                    bool                   `json:"ok"`
	CompressedBytes       int                    `json:"compressed_bytes"`
	InflatedBytes         int                    `json:"inflated_bytes"`
	PayloadIdentical      bool                   `json:"payload_identical"`
	FirstDiffOffset       *int                   `json:"first_diff_offset,omitempty"`
	OriginalPayloadSHA256 string                 `json:"original_payload_sha256"`
	EncodedPayloadSHA256  string                 `json:"encoded_payload_sha256"`
	SectionCount          int                    `json:"section_count"`
	TypedTargets          TypedTargets           `json:"typed_targets"`
	Verification          aoe2.VerificationClaim `json:"verification"`
}

func RoundTripFile(path string) (RoundTripReport, error)

type Section struct {
	Name   string `json:"name"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
	Bytes  int    `json:"bytes"`
	Typed  bool   `json:"typed"`
	SHA256 string `json:"sha256"`
}

type SoundCreateRecipe struct {
	From             int                `json:"from"`
	SoundID          *int16             `json:"sound_id,omitempty"`
	PlayDelay        *int16             `json:"play_delay,omitempty"`
	CacheTime        *int32             `json:"cache_time,omitempty"`
	TotalProbability *int16             `json:"total_probability,omitempty"`
	Items            *[]SoundItemRecipe `json:"items,omitempty"`
}

type SoundDeleteReport struct {
	SoundID                int    `json:"sound_id"`
	Strategy               string `json:"strategy"`
	BeforeTotalProbability int16  `json:"before_total_probability"`
	AfterTotalProbability  int16  `json:"after_total_probability"`
	ItemCount              int    `json:"item_count"`
	Note                   string `json:"note,omitempty"`
}

type SoundItemPatchRecipe struct {
	Index       int     `json:"index"`
	FileName    *string `json:"file_name,omitempty"`
	ResourceID  *int32  `json:"resource_id,omitempty"`
	Probability *int16  `json:"probability,omitempty"`
	Civ         *int16  `json:"civ,omitempty"`
	IconSet     *int16  `json:"icon_set,omitempty"`
}

type SoundItemRecipe struct {
	FileName    string `json:"file_name"`
	ResourceID  int32  `json:"resource_id"`
	Probability int16  `json:"probability"`
	Civ         int16  `json:"civ"`
	IconSet     int16  `json:"icon_set"`
}

type SoundPatchRecipe struct {
	ID               int                    `json:"id"`
	SoundID          *int16                 `json:"sound_id,omitempty"`
	PlayDelay        *int16                 `json:"play_delay,omitempty"`
	CacheTime        *int32                 `json:"cache_time,omitempty"`
	TotalProbability *int16                 `json:"total_probability,omitempty"`
	RemoveItems      []int                  `json:"remove_items,omitempty"`
	Items            []SoundItemPatchRecipe `json:"items,omitempty"`
}

type SoundTarget struct {
	Count    int      `json:"count"`
	Items    int      `json:"items"`
	Editable []string `json:"editable"`
}

type TechCreateRecipe struct {
	From                   int                            `json:"from"`
	Name                   *string                        `json:"name,omitempty"`
	RequiredTechs          []int16                        `json:"required_techs,omitempty"`
	ResourceCosts          []datfile.ResearchResourceCost `json:"resource_costs,omitempty"`
	RequiredTechCount      *int16                         `json:"required_tech_count,omitempty"`
	Civ                    *int16                         `json:"civ,omitempty"`
	FullTechMode           *int16                         `json:"full_tech_mode,omitempty"`
	LanguageDLLName        *int32                         `json:"language_dll_name,omitempty"`
	LanguageDLLDescription *int32                         `json:"language_dll_description,omitempty"`
	EffectID               *int16                         `json:"effect_id,omitempty"`
	Type                   *int16                         `json:"type,omitempty"`
	IconID                 *int16                         `json:"icon_id,omitempty"`
	LanguageDLLHelp        *int32                         `json:"language_dll_help,omitempty"`
	LanguageDLLTechTree    *int32                         `json:"language_dll_tech_tree,omitempty"`
	Repeatable             *uint8                         `json:"repeatable,omitempty"`
	ResearchLocations      []datfile.ResearchLocation     `json:"research_locations,omitempty"`
}

type TechDeleteMigrationReport struct {
	DeleteID              int  `json:"delete_id"`
	TailID                int  `json:"tail_id"`
	CheckedTechIDs        int  `json:"checked_tech_ids"`
	Ready                 bool `json:"ready"`
	CoveredReferences     int  `json:"covered_references"`
	UnsupportedReferences int  `json:"unsupported_references"`
	IDsWithUnsupported    int  `json:"ids_with_unsupported"`
	IDsWithReferences     int  `json:"ids_with_references"`
}

type TechDeleteReport struct {
	TechID            int    `json:"tech_id"`
	Strategy          string `json:"strategy"`
	MovedTailTechID   *int   `json:"moved_tail_tech_id,omitempty"`
	MovedTailTechName string `json:"moved_tail_tech_name,omitempty"`
	ReferenceRewrites int    `json:"reference_rewrites,omitempty"`
	Note              string `json:"note,omitempty"`
}

type TechDisconnectReport struct {
	TechID           int              `json:"tech_id"`
	Complete         bool             `json:"complete"`
	ResidualSummary  ReferenceSummary `json:"residual_summary"`
	Strategy         string           `json:"strategy"`
	RecipeRewrites   int              `json:"recipe_rewrites"`
	StructuralCaveat string           `json:"structural_caveat,omitempty"`
}

type TechExplainReport struct {
	Version      string                 `json:"version"`
	TechID       int                    `json:"tech_id"`
	TechName     string                 `json:"tech_name"`
	EffectID     int16                  `json:"effect_id"`
	EffectName   string                 `json:"effect_name,omitempty"`
	Summary      string                 `json:"summary"`
	TechLines    []string               `json:"tech_lines,omitempty"`
	Effect       *EffectExplainReport   `json:"effect,omitempty"`
	Warnings     []string               `json:"warnings,omitempty"`
	Tech         datfile.Tech           `json:"tech"`
	Verification aoe2.VerificationClaim `json:"verification"`
}

func ExplainTech(idx *datfile.Index, techID int) (TechExplainReport, error)

type TechPatchRecipe struct {
	ID                     int                            `json:"id"`
	Name                   *string                        `json:"name,omitempty"`
	RequiredTechs          []int16                        `json:"required_techs,omitempty"`
	ResourceCosts          []datfile.ResearchResourceCost `json:"resource_costs,omitempty"`
	RequiredTechCount      *int16                         `json:"required_tech_count,omitempty"`
	Civ                    *int16                         `json:"civ,omitempty"`
	FullTechMode           *int16                         `json:"full_tech_mode,omitempty"`
	LanguageDLLName        *int32                         `json:"language_dll_name,omitempty"`
	LanguageDLLDescription *int32                         `json:"language_dll_description,omitempty"`
	EffectID               *int16                         `json:"effect_id,omitempty"`
	Type                   *int16                         `json:"type,omitempty"`
	IconID                 *int16                         `json:"icon_id,omitempty"`
	LanguageDLLHelp        *int32                         `json:"language_dll_help,omitempty"`
	LanguageDLLTechTree    *int32                         `json:"language_dll_tech_tree,omitempty"`
	Repeatable             *uint8                         `json:"repeatable,omitempty"`
	ResearchLocations      []datfile.ResearchLocation     `json:"research_locations,omitempty"`
}

type TechTarget struct {
	Count    int               `json:"count"`
	Spans    []TypedRecordSpan `json:"spans,omitempty"`
	Records  []datfile.Tech    `json:"records,omitempty"`
	Editable []string          `json:"editable"`
}

type TechTreeAvailabilityRef struct {
	Section          string `json:"section"`
	Index            int    `json:"index"`
	Field            string `json:"field"`
	Status           uint8  `json:"status,omitempty"`
	RequiredResearch int32  `json:"required_research,omitempty"`
	EnablingResearch int32  `json:"enabling_research,omitempty"`
	UpperBuilding    int32  `json:"upper_building,omitempty"`
	LocationInAge    int32  `json:"location_in_age,omitempty"`
}

type TechTreeBuildingConnectionCreate struct {
	From             int      `json:"from"`
	ID               *int32   `json:"id,omitempty"`
	Status           *uint8   `json:"status,omitempty"`
	Buildings        *[]int32 `json:"buildings,omitempty"`
	Units            *[]int32 `json:"units,omitempty"`
	Techs            *[]int32 `json:"techs,omitempty"`
	UnitResearch     *[]int32 `json:"unit_research,omitempty"`
	Mode             *[]int32 `json:"mode,omitempty"`
	LocationInAge    *uint8   `json:"location_in_age,omitempty"`
	UnitsTechsTotal  *[]uint8 `json:"units_techs_total,omitempty"`
	UnitsTechsFirst  *[]uint8 `json:"units_techs_first,omitempty"`
	LineMode         *int32   `json:"line_mode,omitempty"`
	EnablingResearch *int32   `json:"enabling_research,omitempty"`
}

type TechTreeBuildingConnectionPatch struct {
	Index            int      `json:"index"`
	ID               *int32   `json:"id,omitempty"`
	Status           *uint8   `json:"status,omitempty"`
	Buildings        *[]int32 `json:"buildings,omitempty"`
	Units            *[]int32 `json:"units,omitempty"`
	Techs            *[]int32 `json:"techs,omitempty"`
	UnitResearch     *[]int32 `json:"unit_research,omitempty"`
	Mode             *[]int32 `json:"mode,omitempty"`
	LocationInAge    *uint8   `json:"location_in_age,omitempty"`
	UnitsTechsTotal  *[]uint8 `json:"units_techs_total,omitempty"`
	UnitsTechsFirst  *[]uint8 `json:"units_techs_first,omitempty"`
	LineMode         *int32   `json:"line_mode,omitempty"`
	EnablingResearch *int32   `json:"enabling_research,omitempty"`
}

type TechTreeIDRewriteRecipe struct {
	Target string `json:"target"`
	From   int32  `json:"from"`
	To     *int32 `json:"to,omitempty"`
	Remove bool   `json:"remove,omitempty"`
}

type TechTreePatchRecipe struct {
	CreateBuildingConnections []TechTreeBuildingConnectionCreate `json:"create_building_connections,omitempty"`
	CreateUnitConnections     []TechTreeUnitConnectionCreate     `json:"create_unit_connections,omitempty"`
	CreateResearchConnections []TechTreeResearchConnectionCreate `json:"create_research_connections,omitempty"`
	DeleteBuildingConnections []int                              `json:"delete_building_connections,omitempty"`
	DeleteUnitConnections     []int                              `json:"delete_unit_connections,omitempty"`
	DeleteResearchConnections []int                              `json:"delete_research_connections,omitempty"`
	BuildingConnections       []TechTreeBuildingConnectionPatch  `json:"building_connections,omitempty"`
	UnitConnections           []TechTreeUnitConnectionPatch      `json:"unit_connections,omitempty"`
	ResearchConnections       []TechTreeResearchConnectionPatch  `json:"research_connections,omitempty"`
	Rewrites                  []TechTreeIDRewriteRecipe          `json:"rewrites,omitempty"`
}

type TechTreeResearchConnectionCreate struct {
	From          int      `json:"from"`
	ID            *int32   `json:"id,omitempty"`
	Status        *uint8   `json:"status,omitempty"`
	UpperBuilding *int32   `json:"upper_building,omitempty"`
	Buildings     *[]int32 `json:"buildings,omitempty"`
	Units         *[]int32 `json:"units,omitempty"`
	Techs         *[]int32 `json:"techs,omitempty"`
	UnitResearch  *[]int32 `json:"unit_research,omitempty"`
	Mode          *[]int32 `json:"mode,omitempty"`
	VerticalLine  *int32   `json:"vertical_line,omitempty"`
	LocationInAge *int32   `json:"location_in_age,omitempty"`
	LineMode      *int32   `json:"line_mode,omitempty"`
}

type TechTreeResearchConnectionPatch struct {
	Index         int      `json:"index"`
	ID            *int32   `json:"id,omitempty"`
	Status        *uint8   `json:"status,omitempty"`
	UpperBuilding *int32   `json:"upper_building,omitempty"`
	Buildings     *[]int32 `json:"buildings,omitempty"`
	Units         *[]int32 `json:"units,omitempty"`
	Techs         *[]int32 `json:"techs,omitempty"`
	UnitResearch  *[]int32 `json:"unit_research,omitempty"`
	Mode          *[]int32 `json:"mode,omitempty"`
	VerticalLine  *int32   `json:"vertical_line,omitempty"`
	LocationInAge *int32   `json:"location_in_age,omitempty"`
	LineMode      *int32   `json:"line_mode,omitempty"`
}

type TechTreeRewriteReport struct {
	Target             string `json:"target"`
	From               int32  `json:"from"`
	To                 *int32 `json:"to,omitempty"`
	Remove             bool   `json:"remove"`
	ReplacedScalars    int    `json:"replaced_scalars"`
	ReplacedListValues int    `json:"replaced_list_values"`
	RemovedListValues  int    `json:"removed_list_values"`
}

type TechTreeUnitConnectionCreate struct {
	From             int      `json:"from"`
	ID               *int32   `json:"id,omitempty"`
	Status           *uint8   `json:"status,omitempty"`
	UpperBuilding    *int32   `json:"upper_building,omitempty"`
	UnitResearch     *[]int32 `json:"unit_research,omitempty"`
	Mode             *[]int32 `json:"mode,omitempty"`
	VerticalLine     *int32   `json:"vertical_line,omitempty"`
	Units            *[]int32 `json:"units,omitempty"`
	LocationInAge    *int32   `json:"location_in_age,omitempty"`
	RequiredResearch *int32   `json:"required_research,omitempty"`
	LineMode         *int32   `json:"line_mode,omitempty"`
	EnablingResearch *int32   `json:"enabling_research,omitempty"`
}

type TechTreeUnitConnectionPatch struct {
	Index            int      `json:"index"`
	ID               *int32   `json:"id,omitempty"`
	Status           *uint8   `json:"status,omitempty"`
	UpperBuilding    *int32   `json:"upper_building,omitempty"`
	UnitResearch     *[]int32 `json:"unit_research,omitempty"`
	Mode             *[]int32 `json:"mode,omitempty"`
	VerticalLine     *int32   `json:"vertical_line,omitempty"`
	Units            *[]int32 `json:"units,omitempty"`
	LocationInAge    *int32   `json:"location_in_age,omitempty"`
	RequiredResearch *int32   `json:"required_research,omitempty"`
	LineMode         *int32   `json:"line_mode,omitempty"`
	EnablingResearch *int32   `json:"enabling_research,omitempty"`
}

type TerrainPassabilityPatch struct {
	TerrainID   int     `json:"terrain_id"`
	Passability float32 `json:"passability"`
}

type TerrainPatchRecipe struct {
	ID              int     `json:"id"`
	Name            *string `json:"name,omitempty"`
	Name2           *string `json:"name_2,omitempty"`
	OverlayMaskName *string `json:"overlay_mask_name,omitempty"`
}

type TerrainRestrictionPatchRecipe struct {
	ID   int                       `json:"id"`
	Rows []TerrainPassabilityPatch `json:"rows,omitempty"`
}

type TerrainRestrictionTarget struct {
	Count           int      `json:"count"`
	TerrainRows     int      `json:"terrain_rows"`
	Editable        []string `json:"editable"`
	PassGraphicNote string   `json:"pass_graphic_note,omitempty"`
}

type TerrainTarget struct {
	Count    int      `json:"count"`
	Editable []string `json:"editable"`
	Note     string   `json:"note,omitempty"`
}

type TrainLocationCard struct {
	Index       int   `json:"index"`
	TrainTime   int16 `json:"train_time"`
	TrainUnitID int16 `json:"train_unit_id"`
	TrainButton uint8 `json:"train_button"`
	TrainHotkey int32 `json:"train_hotkey"`
}

type TypedRecordSpan struct {
	Index  int    `json:"index"`
	Name   string `json:"name,omitempty"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
	Bytes  int    `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type TypedTargets struct {
	Graphics            GraphicTarget            `json:"graphics"`
	Effects             EffectTarget             `json:"effects"`
	UnitHeaders         UnitHeaderTarget         `json:"unit_headers"`
	Techs               TechTarget               `json:"techs"`
	Civs                CivTarget                `json:"civs"`
	Units               UnitTarget               `json:"units"`
	TerrainRestrictions TerrainRestrictionTarget `json:"terrain_restrictions"`
	Terrains            TerrainTarget            `json:"terrains"`
	PlayerColours       PlayerColourTarget       `json:"player_colours"`
	Sounds              SoundTarget              `json:"sounds"`
	RandomMaps          RandomMapTarget          `json:"random_maps"`
}

type UnitAvailabilityCard struct {
	CivID              int                       `json:"civ_id"`
	CivName            string                    `json:"civ_name,omitempty"`
	UnitID             int                       `json:"unit_id"`
	UnitName           string                    `json:"unit_name,omitempty"`
	Present            bool                      `json:"present"`
	Enabled            bool                      `json:"enabled"`
	RawEnabled         uint8                     `json:"raw_enabled,omitempty"`
	Creatable          bool                      `json:"creatable"`
	TrainLocationCount int                       `json:"train_location_count"`
	TrainLocations     []TrainLocationCard       `json:"train_locations,omitempty"`
	TechTreeRefs       []TechTreeAvailabilityRef `json:"tech_tree_refs,omitempty"`
	Status             string                    `json:"status"`
	RecipeHint         json.RawMessage           `json:"recipe_hint,omitempty"`
	Caveats            []string                  `json:"caveats,omitempty"`
}

type UnitAvailabilityCardsReport struct {
	Version      string                  `json:"version"`
	Request      UnitAvailabilityRequest `json:"request"`
	Summary      UnitAvailabilitySummary `json:"summary"`
	Cards        []UnitAvailabilityCard  `json:"cards"`
	Verification aoe2.VerificationClaim  `json:"verification"`
}

func UnitAvailability(idx *datfile.Index, request UnitAvailabilityRequest) (UnitAvailabilityCardsReport, error)

func UnitAvailabilityFile(path string, request UnitAvailabilityRequest) (UnitAvailabilityCardsReport, error)

type UnitAvailabilityRecipe struct {
	CivID   *int  `json:"civ_id,omitempty"`
	CivIDs  []int `json:"civ_ids,omitempty"`
	UnitID  int   `json:"unit_id"`
	AllCivs bool  `json:"all_civs,omitempty"`
	Enabled bool  `json:"enabled"`
}

type UnitAvailabilityReport struct {
	CivID         int    `json:"civ_id"`
	UnitID        int    `json:"unit_id"`
	Present       bool   `json:"present"`
	BeforeEnabled uint8  `json:"before_enabled"`
	AfterEnabled  uint8  `json:"after_enabled"`
	Strategy      string `json:"strategy"`
	Note          string `json:"note,omitempty"`
}

type UnitAvailabilityRequest struct {
	UnitID  int   `json:"unit_id"`
	CivID   *int  `json:"civ_id,omitempty"`
	CivIDs  []int `json:"civ_ids,omitempty"`
	AllCivs bool  `json:"all_civs,omitempty"`
}

type UnitAvailabilitySummary struct {
	CivCount            int `json:"civ_count"`
	Present             int `json:"present"`
	Enabled             int `json:"enabled"`
	Creatable           int `json:"creatable"`
	WithTrainLocations  int `json:"with_train_locations"`
	WithTechTreeRefs    int `json:"with_tech_tree_refs"`
	PlausiblyTrainable  int `json:"plausibly_trainable"`
	Absent              int `json:"absent"`
	Disabled            int `json:"disabled"`
	NeedsMoreInspection int `json:"needs_more_inspection"`
}

type UnitDeleteRecipe struct {
	CivID    *int   `json:"civ_id,omitempty"`
	UnitID   int    `json:"unit_id"`
	AllCivs  bool   `json:"all_civs,omitempty"`
	Strategy string `json:"strategy,omitempty"`
}

type UnitDeleteReport struct {
	CivID         int    `json:"civ_id"`
	UnitID        int    `json:"unit_id"`
	Strategy      string `json:"strategy"`
	BeforeEnabled uint8  `json:"before_enabled"`
	AfterEnabled  uint8  `json:"after_enabled"`
	Note          string `json:"note,omitempty"`
}

type UnitDisconnectReport struct {
	UnitID           int              `json:"unit_id"`
	Complete         bool             `json:"complete"`
	ResidualSummary  ReferenceSummary `json:"residual_summary"`
	Strategy         string           `json:"strategy"`
	RecipeRewrites   int              `json:"recipe_rewrites"`
	StructuralCaveat string           `json:"structural_caveat,omitempty"`
}

type UnitHeaderPatchRecipe struct {
	ID       int                 `json:"id"`
	Tasks    []datfile.TaskPatch `json:"tasks,omitempty"`
	SetTasks *[]datfile.TaskRow  `json:"set_tasks,omitempty"`
}

type UnitHeaderTarget struct {
	Count         int      `json:"count"`
	Present       int      `json:"present"`
	Tasks         int      `json:"tasks"`
	Editable      []string `json:"editable"`
	DecodedFields []string `json:"decoded_fields"`
	Note          string   `json:"note,omitempty"`
}

type UnitTarget struct {
	Count             int      `json:"count"`
	Type50            int      `json:"type50"`
	Creatable         int      `json:"creatable"`
	Buildings         int      `json:"buildings"`
	StorageAttributes int      `json:"storage_attributes"`
	DamageGraphics    int      `json:"damage_graphics"`
	DropSites         int      `json:"drop_sites"`
	Tasks             int      `json:"tasks"`
	Attacks           int      `json:"attacks"`
	Armours           int      `json:"armours"`
	Costs             int      `json:"costs"`
	TrainLocations    int      `json:"train_locations"`
	LinkedBuildings   int      `json:"linked_buildings"`
	TaskRawTailBytes  int      `json:"task_raw_tail_bytes"`
	Editable          []string `json:"editable"`
	Note              string   `json:"note,omitempty"`
}
```

## aoe2kit/pkg/datfacts

```go
package datfacts // import "aoe2kit/pkg/datfacts"


CONSTANTS

const Version = "datfacts.v1"

FUNCTIONS

func ScenarioObjectTypeName(id int) string
func ScenarioUnitAIActionName(id int) string
func ScenarioUnitClassName(id int) string

TYPES

type CatalogReport struct {
	Version      string                 `json:"version"`
	Domain       string                 `json:"domain"`
	Summary      CatalogSummary         `json:"summary"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Rows         []FactRow              `json:"rows"`
}

func Catalog(domain string) (CatalogReport, error)

type CatalogSummary struct {
	Rows               int      `json:"rows"`
	EngineVerified     int      `json:"engine_verified"`
	StrongHypothesis   int      `json:"strong_hypothesis"`
	SourceCorrelated   int      `json:"source_correlated"`
	KnownDisagreements int      `json:"known_disagreements"`
	Sources            []string `json:"sources,omitempty"`
}

type FactRow struct {
	ID            int      `json:"id"`
	Name          string   `json:"name"`
	Meaning       string   `json:"meaning"`
	Family        string   `json:"family,omitempty"`
	Scope         string   `json:"scope,omitempty"`
	Confidence    string   `json:"confidence"`
	Sources       []string `json:"sources,omitempty"`
	Disagreements []string `json:"disagreements,omitempty"`
	SafeUse       string   `json:"safe_use,omitempty"`
	Notes         []string `json:"notes,omitempty"`
}

func ExplainEffectCommandType(id int) (FactRow, error)
```

## aoe2kit/pkg/datfile

```go
package datfile // import "aoe2kit/pkg/datfile"


CONSTANTS

const DebugStringMarker uint16 = 0x0A60

FUNCTIONS

func DecodeUnitFields(record []byte, version string) (map[string]any, error)
    DecodeUnitFields decodes the named scalar and nested fields of a DE unit
    record. It is deliberately map-shaped for the SQL/export boundary: the DAT
    record is a tagged family of layouts, not one fixed row for every unit type.

func Deflate(payload []byte) ([]byte, error)
func EffectAttributeID(name string) (int16, bool)
func EffectAttributeName(attributeID int16) string
func EffectCommandTypeName(commandType uint8) string
func EffectOperationID(name string) (int16, bool)
func EffectOperationName(operationID int16) string
func EffectPackedAttackArmorClassID(name string) (int, bool)
func EffectPackedAttackArmorClassName(classID int) string
func EffectResourceID(name string) (int16, bool)
func EffectResourceName(resourceID int16) string
func EffectTechAttributeID(name string) (int16, bool)
func EffectTechAttributeName(attributeID int16) string
func EffectUnitClassID(name string) (int16, bool)
func EffectUnitClassName(classID int16) string
func EncodeCurrentDETaskList(rows []TaskRow) ([]byte, error)
func ExportSQLite(datPath, outPath string) error
    ExportSQLite writes a query-oriented view of every present DAT unit. Scalar
    decoded fields become columns; variable-length attacks, armours, attributes,
    and damage graphics are child tables. The source DAT remains untouched.

func Inflate(compressed []byte) ([]byte, error)
func RebuildUnitRecord(payload []byte, unit UnitSummary, patch UnitPatch) ([]byte, []string, error)
    RebuildUnitRecord applies a UnitPatch to one parsed unit record and returns
    the rebuilt record bytes.

func RoundTripPayload(payload []byte) error
func UGCAttributeDATField(id int) (string, string)
func UGCAttributeName(id int) string
    UGCAttributeName is the stable public name used by the UGC attribute page.


TYPES

type ActionFieldSpans struct {
	DropSiteCount Span `json:"drop_site_count"`
	DropSites     Span `json:"drop_sites"`
	TaskCount     Span `json:"task_count"`
	Tasks         Span `json:"tasks"`
}

type ActionSummary struct {
	DropSites  []int16          `json:"drop_sites,omitempty"`
	Tasks      []TaskSummary    `json:"tasks,omitempty"`
	FieldSpans ActionFieldSpans `json:"field_spans,omitempty"`
}

type AttributeCost struct {
	Index         int   `json:"index"`
	AttributeType int16 `json:"attribute_type"`
	Amount        int16 `json:"amount"`
	Flag          uint8 `json:"flag"`
	Padding       uint8 `json:"padding"`
	Active        bool  `json:"active"`
	Span          Span  `json:"span"`
}

type AttributeCostPatch struct {
	Index         int    `json:"index"`
	AttributeType *int16 `json:"attribute_type,omitempty"`
	Amount        *int16 `json:"amount,omitempty"`
	Flag          *uint8 `json:"flag,omitempty"`
	Padding       *uint8 `json:"padding,omitempty"`
}

type BuildingConnection struct {
	Index            int            `json:"index"`
	ID               int32          `json:"id"`
	Status           uint8          `json:"status"`
	Buildings        []int32        `json:"buildings,omitempty"`
	Units            []int32        `json:"units,omitempty"`
	Techs            []int32        `json:"techs,omitempty"`
	Common           TechTreeCommon `json:"common"`
	LocationInAge    uint8          `json:"location_in_age"`
	UnitsTechsTotal  []uint8        `json:"units_techs_total,omitempty"`
	UnitsTechsFirst  []uint8        `json:"units_techs_first,omitempty"`
	LineMode         int32          `json:"line_mode"`
	EnablingResearch int32          `json:"enabling_research"`
	Span             Span           `json:"span"`
}

type BuildingSummary struct {
	SpawningGraphic    int16            `json:"spawning_graphic,omitempty"`
	UpgradeGraphic     int16            `json:"upgrade_graphic,omitempty"`
	StackUnit          int16            `json:"stack_unit,omitempty"`
	HeadUnit           int16            `json:"head_unit,omitempty"`
	TransformUnit      int16            `json:"transform_unit,omitempty"`
	PileUnit           int16            `json:"pile_unit,omitempty"`
	ConstructionUnit   int16            `json:"construction_unit,omitempty"`
	SalvageUnit        int16            `json:"salvage_unit,omitempty"`
	TransformSound     int16            `json:"transform_sound,omitempty"`
	ConstructionSound  int16            `json:"construction_sound,omitempty"`
	GarrisonType       uint8            `json:"garrison_type,omitempty"`
	GarrisonHealRate   float32          `json:"garrison_heal_rate,omitempty"`
	GarrisonRepairRate float32          `json:"garrison_repair_rate,omitempty"`
	SalvageAttributes  []uint8          `json:"salvage_attributes,omitempty"`
	LinkedBuildings    []LinkedBuilding `json:"linked_buildings,omitempty"`
}

type Civ struct {
	Index         int           `json:"index"`
	Type          uint8         `json:"type"`
	Name          string        `json:"name"`
	ResourcesSize int           `json:"resources_size"`
	TechTreeID    int16         `json:"tech_tree_id"`
	TeamBonusID   int16         `json:"team_bonus_id"`
	Resources     []float32     `json:"resources,omitempty"`
	IconSet       uint8         `json:"icon_set"`
	UnitsSize     int           `json:"units_size"`
	Units         []UnitSummary `json:"units,omitempty"`
	FieldSpans    CivFieldSpans `json:"field_spans,omitempty"`
	Span          Span          `json:"span"`
}

type CivFieldSpans struct {
	Type          Span `json:"type"`
	Name          Span `json:"name"`
	ResourcesSize Span `json:"resources_size"`
	TechTreeID    Span `json:"tech_tree_id"`
	TeamBonusID   Span `json:"team_bonus_id"`
	Resources     Span `json:"resources"`
	IconSet       Span `json:"icon_set"`
	UnitsSize     Span `json:"units_size"`
	UnitPointers  Span `json:"unit_pointers"`
}

type CreatableFieldSpans struct {
	TrainLocationCount Span `json:"train_location_count"`
	TrainLocations     Span `json:"train_locations"`
	TrainTime0         Span `json:"train_time_0"`
	TrainUnitID0       Span `json:"train_unit_id_0"`
	TrainButtonID0     Span `json:"train_button_id_0"`
	TrainHotKeyID0     Span `json:"train_hotkey_id_0"`
	ButtonIconID       Span `json:"button_icon_id"`
	ButtonHotkeyAction Span `json:"button_hotkey_action"`
}

type CreatableSummary struct {
	TrainLocationCount int                 `json:"train_location_count"`
	TrainTime0         int16               `json:"train_time_0"`
	TrainUnitID0       int16               `json:"train_unit_id_0"`
	TrainButtonID0     uint8               `json:"train_button_id_0"`
	TrainHotKeyID0     int32               `json:"train_hotkey_id_0"`
	ButtonIconID       int16               `json:"button_icon_id"`
	ButtonHotkeyAction int16               `json:"button_hotkey_action"`
	Costs              []AttributeCost     `json:"costs,omitempty"`
	TrainLocations     []TrainLocation     `json:"train_locations,omitempty"`
	FieldSpans         CreatableFieldSpans `json:"field_spans,omitempty"`
}

type DamageGraphic struct {
	Index         int    `json:"index"`
	GraphicID     uint16 `json:"graphic_id"`
	DamagePercent uint16 `json:"damage_percent"`
	Flag          uint8  `json:"flag"`
	Span          Span   `json:"span"`
}

type DamageGraphicPatch struct {
	Index         int     `json:"index"`
	GraphicID     *uint16 `json:"graphic_id,omitempty"`
	DamagePercent *uint16 `json:"damage_percent,omitempty"`
	Flag          *uint8  `json:"flag,omitempty"`
}

type DamageGraphicRow struct {
	GraphicID     uint16 `json:"graphic_id"`
	DamagePercent uint16 `json:"damage_percent"`
	Flag          uint8  `json:"flag"`
}

type DebugString struct {
	Value      string `json:"value"`
	Marker     uint16 `json:"marker"`
	ByteLength int    `json:"byte_length"`
	Span       Span   `json:"span"`
	DataSpan   Span   `json:"data_span"`
}

type DiffEntry struct {
	Kind       string `json:"kind"`
	Section    string `json:"section"`
	Key        string `json:"key"`
	Name       string `json:"name,omitempty"`
	BaseSHA256 string `json:"base_sha256,omitempty"`
	ModSHA256  string `json:"mod_sha256,omitempty"`
}

type DiffOptions struct {
	Limit    int
	All      bool
	Sections []string
}

type DiffReport struct {
	Base         string        `json:"base"`
	Mod          string        `json:"mod"`
	Verification string        `json:"verification"`
	Same         bool          `json:"same"`
	Summary      DiffSummary   `json:"summary"`
	Sections     []DiffSection `json:"sections,omitempty"`
	Warnings     []string      `json:"warnings,omitempty"`
}

func DiffFiles(basePath, modPath string, opts DiffOptions) (*DiffReport, error)

type DiffSection struct {
	Name      string      `json:"name"`
	BaseCount int         `json:"base_count"`
	ModCount  int         `json:"mod_count"`
	Added     int         `json:"added"`
	Removed   int         `json:"removed"`
	Changed   int         `json:"changed"`
	Entries   []DiffEntry `json:"entries,omitempty"`
	Truncated bool        `json:"truncated,omitempty"`
}

type DiffSummary struct {
	SectionsCompared int  `json:"sections_compared"`
	Added            int  `json:"added"`
	Removed          int  `json:"removed"`
	Changed          int  `json:"changed"`
	Emitted          int  `json:"emitted"`
	Limit            int  `json:"limit"`
	Truncated        bool `json:"truncated"`
}

type Effect struct {
	Index       int             `json:"index"`
	Name        string          `json:"name"`
	CommandSize int             `json:"command_size"`
	Commands    []EffectCommand `json:"commands,omitempty"`
	Span        Span            `json:"span"`
}

type EffectCommand struct {
	Index    int                    `json:"index"`
	Type     uint8                  `json:"type"`
	A        int16                  `json:"a"`
	B        int16                  `json:"b"`
	C        int16                  `json:"c"`
	D        float32                `json:"d"`
	Semantic *EffectCommandSemantic `json:"semantic,omitempty"`
	Span     Span                   `json:"span"`
}

type EffectCommandReference struct {
	Kind       string `json:"kind"`
	Field      string `json:"field"`
	ID         int    `json:"id"`
	Confidence string `json:"confidence"`
}

type EffectCommandSemantic struct {
	TypeName          string                   `json:"type_name"`
	Scope             string                   `json:"scope,omitempty"`
	TargetUnit        int16                    `json:"target_unit"`
	UnitClassID       int16                    `json:"unit_class_id"`
	UnitClassName     string                   `json:"unit_class_name,omitempty"`
	AttributeID       int16                    `json:"attribute_id"`
	AttributeName     string                   `json:"attribute_name,omitempty"`
	PackedTypeID      *int                     `json:"packed_type_id,omitempty"`
	PackedTypeName    string                   `json:"packed_type_name,omitempty"`
	PackedAmount      *int                     `json:"packed_amount,omitempty"`
	StringID          *int                     `json:"string_id,omitempty"`
	ResourceID        *int16                   `json:"resource_id,omitempty"`
	ResourceName      string                   `json:"resource_name,omitempty"`
	OperationID       *int16                   `json:"operation_id,omitempty"`
	OperationName     string                   `json:"operation_name,omitempty"`
	TechAttributeID   *int16                   `json:"tech_attribute_id,omitempty"`
	TechAttributeName string                   `json:"tech_attribute_name,omitempty"`
	Amount            float32                  `json:"amount"`
	ReferenceKind     string                   `json:"reference_kind,omitempty"`
	ReferenceField    string                   `json:"reference_field,omitempty"`
	ReferenceID       *int                     `json:"reference_id,omitempty"`
	References        []EffectCommandReference `json:"references,omitempty"`
	Confidence        string                   `json:"confidence"`
	Source            string                   `json:"source"`
}

func InterpretEffectCommand(command EffectCommand) *EffectCommandSemantic

type Graphic struct {
	Index              int                 `json:"index"`
	Present            bool                `json:"present"`
	Name               DebugString         `json:"name"`
	FileName           DebugString         `json:"file_name"`
	ParticleEffectName DebugString         `json:"particle_effect_name"`
	SLP                int32               `json:"slp"`
	IsLoaded           int8                `json:"is_loaded"`
	OldColorFlag       int8                `json:"old_color_flag"`
	Layer              int8                `json:"layer"`
	PlayerColor        int8                `json:"player_color"`
	Rainbow            int8                `json:"rainbow"`
	TransparentSelect  int8                `json:"transparent_selection"`
	Coordinates        [4]int16            `json:"coordinates"`
	SoundID            int16               `json:"sound_id"`
	WwiseSoundID       uint32              `json:"wwise_sound_id"`
	AngleSoundsUsed    int8                `json:"angle_sounds_used"`
	FrameCount         int16               `json:"frame_count"`
	AngleCount         int16               `json:"angle_count"`
	DeltaCount         int16               `json:"delta_count"`
	SpeedMultiplier    float32             `json:"speed_multiplier"`
	FrameDuration      float32             `json:"frame_duration"`
	ReplayDelay        float32             `json:"replay_delay"`
	SequenceType       uint8               `json:"sequence_type"`
	ID                 int16               `json:"id"`
	MirroringMode      int8                `json:"mirroring_mode"`
	EditorFlag         int8                `json:"editor_flag"`
	Deltas             []GraphicDelta      `json:"deltas,omitempty"`
	AngleSounds        []GraphicAngleSound `json:"angle_sounds,omitempty"`
	FieldSpans         GraphicFieldSpans   `json:"field_spans,omitempty"`
	RecordSpan         Span                `json:"record_span"`
}

func (g Graphic) Summary() GraphicSummary

type GraphicAngleSound struct {
	Index         int    `json:"index"`
	FrameNum      int16  `json:"frame_num"`
	SoundID       int16  `json:"sound_id"`
	WwiseSoundID  uint32 `json:"wwise_sound_id"`
	FrameNum2     int16  `json:"frame_num_2"`
	WwiseSoundID2 uint32 `json:"wwise_sound_id_2"`
	SoundID2      int16  `json:"sound_id_2"`
	FrameNum3     int16  `json:"frame_num_3"`
	WwiseSoundID3 uint32 `json:"wwise_sound_id_3"`
	SoundID3      int16  `json:"sound_id_3"`
	Span          Span   `json:"span"`
}

type GraphicAngleSoundRow struct {
	FrameNum      int16  `json:"frame_num"`
	SoundID       int16  `json:"sound_id"`
	WwiseSoundID  uint32 `json:"wwise_sound_id"`
	FrameNum2     int16  `json:"frame_num_2"`
	WwiseSoundID2 uint32 `json:"wwise_sound_id_2"`
	SoundID2      int16  `json:"sound_id_2"`
	FrameNum3     int16  `json:"frame_num_3"`
	WwiseSoundID3 uint32 `json:"wwise_sound_id_3"`
	SoundID3      int16  `json:"sound_id_3"`
}

type GraphicCreateRecipe struct {
	From               int                     `json:"from"`
	Name               *string                 `json:"name,omitempty"`
	FileName           *string                 `json:"file_name,omitempty"`
	ParticleEffectName *string                 `json:"particle_effect_name,omitempty"`
	SLP                *int32                  `json:"slp,omitempty"`
	IsLoaded           *int8                   `json:"is_loaded,omitempty"`
	OldColorFlag       *int8                   `json:"old_color_flag,omitempty"`
	Layer              *int8                   `json:"layer,omitempty"`
	PlayerColor        *int8                   `json:"player_color,omitempty"`
	Rainbow            *int8                   `json:"rainbow,omitempty"`
	TransparentSelect  *int8                   `json:"transparent_selection,omitempty"`
	Coordinates        []int16                 `json:"coordinates,omitempty"`
	SoundID            *int16                  `json:"sound_id,omitempty"`
	WwiseSoundID       *uint32                 `json:"wwise_sound_id,omitempty"`
	FrameCount         *int16                  `json:"frame_count,omitempty"`
	SpeedMultiplier    *float32                `json:"speed_multiplier,omitempty"`
	FrameDuration      *float32                `json:"frame_duration,omitempty"`
	ReplayDelay        *float32                `json:"replay_delay,omitempty"`
	SequenceType       *uint8                  `json:"sequence_type,omitempty"`
	MirroringMode      *int8                   `json:"mirroring_mode,omitempty"`
	EditorFlag         *int8                   `json:"editor_flag,omitempty"`
	SetDeltas          *[]GraphicDeltaRow      `json:"set_deltas,omitempty"`
	SetAngleSounds     *[]GraphicAngleSoundRow `json:"set_angle_sounds,omitempty"`
}

type GraphicCreateReport struct {
	InputCompressedBytes  int                    `json:"input_compressed_bytes"`
	OutputCompressedBytes int                    `json:"output_compressed_bytes"`
	InputInflatedBytes    int                    `json:"input_inflated_bytes"`
	OutputInflatedBytes   int                    `json:"output_inflated_bytes"`
	InflatedLengthDelta   int                    `json:"inflated_length_delta"`
	FromGraphicID         int                    `json:"from_graphic_id"`
	NewGraphicID          int                    `json:"new_graphic_id"`
	BeforeGraphicsSize    int                    `json:"before_graphics_size"`
	AfterGraphicsSize     int                    `json:"after_graphics_size"`
	Template              GraphicSummary         `json:"template"`
	Created               GraphicSummary         `json:"created"`
	ChangedFields         []string               `json:"changed_fields"`
	Verified              bool                   `json:"verified"`
	Verification          aoe2.VerificationClaim `json:"verification"`
}

func CreateGraphic(compressed []byte, recipe GraphicCreateRecipe) ([]byte, GraphicCreateReport, error)

type GraphicDeleteReport struct {
	InputCompressedBytes  int                    `json:"input_compressed_bytes"`
	OutputCompressedBytes int                    `json:"output_compressed_bytes"`
	InputInflatedBytes    int                    `json:"input_inflated_bytes"`
	OutputInflatedBytes   int                    `json:"output_inflated_bytes"`
	InflatedLengthDelta   int                    `json:"inflated_length_delta"`
	GraphicID             int                    `json:"graphic_id"`
	BeforeGraphicsSize    int                    `json:"before_graphics_size"`
	AfterGraphicsSize     int                    `json:"after_graphics_size"`
	RemovedRecordBytes    int                    `json:"removed_record_bytes"`
	Before                GraphicSummary         `json:"before"`
	Verified              bool                   `json:"verified"`
	Verification          aoe2.VerificationClaim `json:"verification"`
}

func DeleteGraphic(compressed []byte, graphicID int) ([]byte, GraphicDeleteReport, error)

type GraphicDelta struct {
	Index        int   `json:"index"`
	GraphicID    int16 `json:"graphic_id"`
	Padding1     int16 `json:"padding_1"`
	SpritePtr    int32 `json:"sprite_ptr"`
	OffsetX      int16 `json:"offset_x"`
	OffsetY      int16 `json:"offset_y"`
	DisplayAngle int16 `json:"display_angle"`
	Padding2     int16 `json:"padding_2"`
	Span         Span  `json:"span"`
}

type GraphicDeltaRow struct {
	GraphicID    int16 `json:"graphic_id"`
	Padding1     int16 `json:"padding_1"`
	SpritePtr    int32 `json:"sprite_ptr"`
	OffsetX      int16 `json:"offset_x"`
	OffsetY      int16 `json:"offset_y"`
	DisplayAngle int16 `json:"display_angle"`
	Padding2     int16 `json:"padding_2"`
}

type GraphicFieldSpans struct {
	SLP               Span    `json:"slp"`
	IsLoaded          Span    `json:"is_loaded"`
	OldColorFlag      Span    `json:"old_color_flag"`
	Layer             Span    `json:"layer"`
	PlayerColor       Span    `json:"player_color"`
	Rainbow           Span    `json:"rainbow"`
	TransparentSelect Span    `json:"transparent_selection"`
	Coordinates       [4]Span `json:"coordinates"`
	DeltaCount        Span    `json:"delta_count"`
	SoundID           Span    `json:"sound_id"`
	WwiseSoundID      Span    `json:"wwise_sound_id"`
	AngleSoundsUsed   Span    `json:"angle_sounds_used"`
	FrameCount        Span    `json:"frame_count"`
	AngleCount        Span    `json:"angle_count"`
	SpeedMultiplier   Span    `json:"speed_multiplier"`
	FrameDuration     Span    `json:"frame_duration"`
	ReplayDelay       Span    `json:"replay_delay"`
	SequenceType      Span    `json:"sequence_type"`
	ID                Span    `json:"id"`
	MirroringMode     Span    `json:"mirroring_mode"`
	EditorFlag        Span    `json:"editor_flag"`
}

type GraphicFilter struct {
	NameContains string
	ParticleOnly bool
}

type GraphicPatch struct {
	Name               *string                 `json:"name,omitempty"`
	FileName           *string                 `json:"file_name,omitempty"`
	ParticleEffectName *string                 `json:"particle_effect_name,omitempty"`
	SLP                *int32                  `json:"slp,omitempty"`
	IsLoaded           *int8                   `json:"is_loaded,omitempty"`
	OldColorFlag       *int8                   `json:"old_color_flag,omitempty"`
	Layer              *int8                   `json:"layer,omitempty"`
	PlayerColor        *int8                   `json:"player_color,omitempty"`
	Rainbow            *int8                   `json:"rainbow,omitempty"`
	TransparentSelect  *int8                   `json:"transparent_selection,omitempty"`
	Coordinates        []int16                 `json:"coordinates,omitempty"`
	SoundID            *int16                  `json:"sound_id,omitempty"`
	WwiseSoundID       *uint32                 `json:"wwise_sound_id,omitempty"`
	FrameCount         *int16                  `json:"frame_count,omitempty"`
	SpeedMultiplier    *float32                `json:"speed_multiplier,omitempty"`
	FrameDuration      *float32                `json:"frame_duration,omitempty"`
	ReplayDelay        *float32                `json:"replay_delay,omitempty"`
	SequenceType       *uint8                  `json:"sequence_type,omitempty"`
	MirroringMode      *int8                   `json:"mirroring_mode,omitempty"`
	EditorFlag         *int8                   `json:"editor_flag,omitempty"`
	SetDeltas          *[]GraphicDeltaRow      `json:"set_deltas,omitempty"`
	SetAngleSounds     *[]GraphicAngleSoundRow `json:"set_angle_sounds,omitempty"`
}

func (patch GraphicPatch) Empty() bool

type GraphicRecipePatch struct {
	ID                 int                     `json:"id"`
	Name               *string                 `json:"name,omitempty"`
	FileName           *string                 `json:"file_name,omitempty"`
	ParticleEffectName *string                 `json:"particle_effect_name,omitempty"`
	SLP                *int32                  `json:"slp,omitempty"`
	IsLoaded           *int8                   `json:"is_loaded,omitempty"`
	OldColorFlag       *int8                   `json:"old_color_flag,omitempty"`
	Layer              *int8                   `json:"layer,omitempty"`
	PlayerColor        *int8                   `json:"player_color,omitempty"`
	Rainbow            *int8                   `json:"rainbow,omitempty"`
	TransparentSelect  *int8                   `json:"transparent_selection,omitempty"`
	Coordinates        []int16                 `json:"coordinates,omitempty"`
	SoundID            *int16                  `json:"sound_id,omitempty"`
	WwiseSoundID       *uint32                 `json:"wwise_sound_id,omitempty"`
	FrameCount         *int16                  `json:"frame_count,omitempty"`
	SpeedMultiplier    *float32                `json:"speed_multiplier,omitempty"`
	FrameDuration      *float32                `json:"frame_duration,omitempty"`
	ReplayDelay        *float32                `json:"replay_delay,omitempty"`
	SequenceType       *uint8                  `json:"sequence_type,omitempty"`
	MirroringMode      *int8                   `json:"mirroring_mode,omitempty"`
	EditorFlag         *int8                   `json:"editor_flag,omitempty"`
	SetDeltas          *[]GraphicDeltaRow      `json:"set_deltas,omitempty"`
	SetAngleSounds     *[]GraphicAngleSoundRow `json:"set_angle_sounds,omitempty"`
}

type GraphicSummary struct {
	Index              int     `json:"index"`
	Name               string  `json:"name"`
	FileName           string  `json:"file_name"`
	ParticleEffectName string  `json:"particle_effect_name"`
	SLP                int32   `json:"slp"`
	Layer              int8    `json:"layer"`
	PlayerColor        int8    `json:"player_color"`
	SoundID            int16   `json:"sound_id"`
	WwiseSoundID       uint32  `json:"wwise_sound_id"`
	FrameCount         int16   `json:"frame_count"`
	AngleCount         int16   `json:"angle_count"`
	DeltaCount         int16   `json:"delta_count"`
	AngleSoundCount    int     `json:"angle_sound_count"`
	SpeedMultiplier    float32 `json:"speed_multiplier"`
	FrameDuration      float32 `json:"frame_duration"`
	ReplayDelay        float32 `json:"replay_delay"`
	SequenceType       uint8   `json:"sequence_type"`
	ID                 int16   `json:"id"`
	RecordStart        int     `json:"record_start"`
	RecordEnd          int     `json:"record_end"`
}

type Index struct {
	Path                string               `json:"path,omitempty"`
	Compressed          int                  `json:"compressed_bytes"`
	Inflated            int                  `json:"inflated_bytes"`
	Version             string               `json:"version"`
	GraphicsSize        int                  `json:"graphics_size"`
	Graphics            []Graphic            `json:"graphics,omitempty"`
	Effects             []Effect             `json:"effects,omitempty"`
	UnitHeaders         []UnitHeader         `json:"unit_headers,omitempty"`
	Civs                []Civ                `json:"civs,omitempty"`
	Techs               []Tech               `json:"techs,omitempty"`
	TerrainRestrictions []TerrainRestriction `json:"terrain_restrictions,omitempty"`
	Terrains            []Terrain            `json:"terrains,omitempty"`
	PlayerColours       []PlayerColour       `json:"player_colours,omitempty"`
	Sounds              []Sound              `json:"sounds,omitempty"`
	RandomMaps          []RandomMapInfo      `json:"random_maps,omitempty"`
	GameMetrics         Metrics              `json:"game_metrics,omitempty"`
	TechTree            TechTree             `json:"tech_tree,omitempty"`
	Spans               []Span               `json:"spans,omitempty"`
}

func Open(path string) (*Index, error)

func Parse(payload []byte) (*Index, error)

func (idx *Index) Effect(id int) (Effect, bool)

func (idx *Index) Graphic(id int) (Graphic, bool)

func (idx *Index) Palette(opts PaletteOptions) PaletteReport

func (idx *Index) PresentGraphicSummaries() []GraphicSummary

func (idx *Index) PresentGraphicSummariesFiltered(filter GraphicFilter) []GraphicSummary

func (idx *Index) PresentGraphics() []Graphic

func (idx *Index) PresentGraphicsFiltered(filter GraphicFilter) []Graphic

func (idx *Index) Tech(id int) (Tech, bool)

func (idx *Index) ValidateCoverage(payload []byte) error

type LinkedBuilding struct {
	Index  int     `json:"index"`
	UnitID int16   `json:"unit_id"`
	X      float32 `json:"x"`
	Y      float32 `json:"y"`
	Active bool    `json:"active"`
	Span   Span    `json:"span"`
}

type LinkedBuildingPatch struct {
	Index  int      `json:"index"`
	UnitID *int16   `json:"unit_id,omitempty"`
	X      *float32 `json:"x,omitempty"`
	Y      *float32 `json:"y,omitempty"`
}

type Metrics struct {
	TimeSlice         int32 `json:"time_slice"`
	UnitKillRate      int32 `json:"unit_kill_rate"`
	UnitKillTotal     int32 `json:"unit_kill_total"`
	UnitHitPointRate  int32 `json:"unit_hit_point_rate"`
	UnitHitPointTotal int32 `json:"unit_hit_point_total"`
	RazingKillRate    int32 `json:"razing_kill_rate"`
	RazingKillTotal   int32 `json:"razing_kill_total"`
	Span              Span  `json:"span"`
}

type PaletteOptions struct {
	ID          *int
	MinVariants int
}

type PaletteReport struct {
	Path         string         `json:"path,omitempty"`
	Version      string         `json:"version"`
	UnitCount    int            `json:"unit_count"`
	Returned     int            `json:"returned"`
	Filters      PaletteOptions `json:"filters"`
	Verification string         `json:"verification"`
	Rows         []PaletteRow   `json:"rows"`
}

func PaletteFile(path string, opts PaletteOptions) (PaletteReport, error)

type PaletteRow struct {
	UnitID           int    `json:"unit_id"`
	UnitName         string `json:"unit_name"`
	CivIndices       []int  `json:"civ_indices,omitempty"`
	UnitType         int    `json:"unit_type"`
	UnitClass        int16  `json:"unit_class"`
	UnitClassName    string `json:"unit_class_name,omitempty"`
	StandingGraphic1 int    `json:"standing_graphic_1"`
	GraphicName      string `json:"graphic_name,omitempty"`
	FileName         string `json:"file_name,omitempty"`
	SLP              int32  `json:"slp"`
	AngleCount       int    `json:"angle_count"`
	FrameCount       int    `json:"frame_count"`
	SequenceType     uint8  `json:"sequence_type"`
	VariantCount     *int   `json:"variant_count,omitempty"`
	VariantNote      string `json:"variant_note,omitempty"`
	Classification   string `json:"classification"`
	Confidence       string `json:"confidence"`
	Evidence         string `json:"evidence"`
	RotationEncoding string `json:"rotation_encoding,omitempty"`
}

type PatchReport struct {
	InputCompressedBytes  int                    `json:"input_compressed_bytes"`
	OutputCompressedBytes int                    `json:"output_compressed_bytes"`
	InputInflatedBytes    int                    `json:"input_inflated_bytes"`
	OutputInflatedBytes   int                    `json:"output_inflated_bytes"`
	InflatedLengthDelta   int                    `json:"inflated_length_delta"`
	GraphicID             int                    `json:"graphic_id"`
	Before                GraphicSummary         `json:"before"`
	After                 GraphicSummary         `json:"after"`
	ChangedFields         []string               `json:"changed_fields"`
	Verified              bool                   `json:"verified"`
	Verification          aoe2.VerificationClaim `json:"verification"`
}

func PatchGraphic(compressed []byte, graphicID int, patch GraphicPatch) ([]byte, PatchReport, error)

func PatchGraphicFile(inputPath, outputPath string, graphicID int, patch GraphicPatch) (PatchReport, error)

type PlayerColour struct {
	Index               int   `json:"index"`
	ID                  int32 `json:"id"`
	Base                int32 `json:"base"`
	UnitOutlineColour   int32 `json:"unit_outline_colour"`
	SelectionColour1    int32 `json:"selection_colour_1"`
	SelectionColour2    int32 `json:"selection_colour_2"`
	MinimapColour1      int32 `json:"minimap_colour_1"`
	MinimapColour2      int32 `json:"minimap_colour_2"`
	MinimapColour3      int32 `json:"minimap_colour_3"`
	StatisticsTextColor int32 `json:"statistics_text_color"`
	Span                Span  `json:"span"`
}

type RandomMapInfo struct {
	Pass          int    `json:"pass"`
	Index         int    `json:"index"`
	Lands         uint32 `json:"lands"`
	Terrains      uint32 `json:"terrains"`
	Units         uint32 `json:"units"`
	Elevations    uint32 `json:"elevations"`
	Span          Span   `json:"span"`
	LandSpan      Span   `json:"land_span"`
	TerrainSpan   Span   `json:"terrain_span"`
	UnitSpan      Span   `json:"unit_span"`
	ElevationSpan Span   `json:"elevation_span"`
}

type Recipe struct {
	Graphics       []GraphicRecipePatch  `json:"graphics,omitempty"`
	CreateGraphic  *GraphicCreateRecipe  `json:"create_graphic,omitempty"`
	CreateGraphics []GraphicCreateRecipe `json:"create_graphics,omitempty"`
	DeleteGraphics []int                 `json:"delete_graphics,omitempty"`
	CreateUnit     *UnitCreateRecipe     `json:"create_unit,omitempty"`
	CreateUnits    []UnitCreateRecipe    `json:"create_units,omitempty"`
	Units          []UnitRecipePatch     `json:"units,omitempty"`
}

func (recipe Recipe) Empty() bool

type RecipeReport struct {
	InputCompressedBytes  int                    `json:"input_compressed_bytes"`
	OutputCompressedBytes int                    `json:"output_compressed_bytes"`
	InputInflatedBytes    int                    `json:"input_inflated_bytes"`
	OutputInflatedBytes   int                    `json:"output_inflated_bytes"`
	InflatedLengthDelta   int                    `json:"inflated_length_delta"`
	GraphicReports        []PatchReport          `json:"graphic_reports,omitempty"`
	CreatedGraphics       []GraphicCreateReport  `json:"created_graphics,omitempty"`
	DeletedGraphics       []GraphicDeleteReport  `json:"deleted_graphics,omitempty"`
	CreatedUnits          []UnitCreateReport     `json:"created_units,omitempty"`
	UnitReports           []UnitPatchReport      `json:"unit_reports,omitempty"`
	Verified              bool                   `json:"verified"`
	Verification          aoe2.VerificationClaim `json:"verification"`
}

func PatchRecipe(compressed []byte, recipe Recipe) ([]byte, RecipeReport, error)

func PatchRecipeFile(inputPath, outputPath string, recipe Recipe) (RecipeReport, error)

func PlanRecipeFile(inputPath string, recipe Recipe) (RecipeReport, error)

type ResearchConnection struct {
	Index         int            `json:"index"`
	ID            int32          `json:"id"`
	Status        uint8          `json:"status"`
	UpperBuilding int32          `json:"upper_building"`
	Buildings     []int32        `json:"buildings,omitempty"`
	Units         []int32        `json:"units,omitempty"`
	Techs         []int32        `json:"techs,omitempty"`
	Common        TechTreeCommon `json:"common"`
	VerticalLine  int32          `json:"vertical_line"`
	LocationInAge int32          `json:"location_in_age"`
	LineMode      int32          `json:"line_mode"`
	Span          Span           `json:"span"`
}

type ResearchLocation struct {
	LocationID   int16 `json:"location_id"`
	ResearchTime int16 `json:"research_time"`
	ButtonID     uint8 `json:"button_id"`
	HotKeyID     int32 `json:"hotkey_id"`
}

type ResearchResourceCost struct {
	Type   int16 `json:"type"`
	Amount int16 `json:"amount"`
	Flag   uint8 `json:"flag"`
}

type Sound struct {
	Index            int         `json:"index"`
	ID               int16       `json:"id"`
	PlayDelay        int16       `json:"play_delay"`
	CacheTime        int32       `json:"cache_time"`
	TotalProbability int16       `json:"total_probability"`
	Items            []SoundItem `json:"items,omitempty"`
	Span             Span        `json:"span"`
}

type SoundItem struct {
	Index       int         `json:"index"`
	FileName    DebugString `json:"file_name"`
	ResourceID  int32       `json:"resource_id"`
	Probability int16       `json:"probability"`
	Civ         int16       `json:"civ"`
	IconSet     int16       `json:"icon_set"`
	Span        Span        `json:"span"`
}

type Span struct {
	Name  string `json:"name"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

func (s Span) Len() int

type TaskPatch struct {
	Index             int      `json:"index"`
	RecordType        *int16   `json:"record_type,omitempty"`
	ID                *int16   `json:"id,omitempty"`
	IsDefault         *uint8   `json:"is_default,omitempty"`
	ActionType        *int16   `json:"action_type,omitempty"`
	ObjectClass       *int16   `json:"object_class,omitempty"`
	ObjectID          *int16   `json:"object_id,omitempty"`
	TerrainID         *int16   `json:"terrain_id,omitempty"`
	AttributeTypes    []int16  `json:"attribute_types,omitempty"`
	WorkValue1        *float32 `json:"work_value_1,omitempty"`
	WorkValue2        *float32 `json:"work_value_2,omitempty"`
	WorkRange         *float32 `json:"work_range,omitempty"`
	AutoSearchTargets *uint8   `json:"auto_search_targets,omitempty"`
	SearchWaitTime    *float32 `json:"search_wait_time,omitempty"`
	EnableTargeting   *uint8   `json:"enable_targeting,omitempty"`
	CombatLevel       *uint8   `json:"combat_level,omitempty"`
}

type TaskRow struct {
	RecordType        int16   `json:"record_type"`
	ID                int16   `json:"id"`
	IsDefault         uint8   `json:"is_default"`
	ActionType        int16   `json:"action_type"`
	ObjectClass       int16   `json:"object_class"`
	ObjectID          int16   `json:"object_id"`
	TerrainID         int16   `json:"terrain_id"`
	AttributeTypes    []int16 `json:"attribute_types"`
	WorkValue1        float32 `json:"work_value_1"`
	WorkValue2        float32 `json:"work_value_2"`
	WorkRange         float32 `json:"work_range"`
	AutoSearchTargets uint8   `json:"auto_search_targets"`
	SearchWaitTime    float32 `json:"search_wait_time"`
	EnableTargeting   uint8   `json:"enable_targeting"`
	CombatLevel       uint8   `json:"combat_level"`
	RawTail           []byte  `json:"raw_tail"`
}

type TaskSummary struct {
	Index             int     `json:"index"`
	RecordType        int16   `json:"record_type"`
	ID                int16   `json:"id"`
	IsDefault         uint8   `json:"is_default"`
	ActionType        int16   `json:"action_type"`
	ObjectClass       int16   `json:"object_class"`
	ObjectID          int16   `json:"object_id"`
	TerrainID         int16   `json:"terrain_id"`
	AttributeTypes    []int16 `json:"attribute_types,omitempty"`
	WorkValue1        float32 `json:"work_value_1"`
	WorkValue2        float32 `json:"work_value_2"`
	WorkRange         float32 `json:"work_range"`
	AutoSearchTargets uint8   `json:"auto_search_targets"`
	SearchWaitTime    float32 `json:"search_wait_time"`
	EnableTargeting   uint8   `json:"enable_targeting"`
	CombatLevel       uint8   `json:"combat_level"`
	RawTail           []byte  `json:"raw_tail,omitempty"`
	Span              Span    `json:"span"`
}

type Tech struct {
	Index                  int                    `json:"index"`
	RequiredTechs          []int16                `json:"required_techs,omitempty"`
	ResourceCosts          []ResearchResourceCost `json:"resource_costs,omitempty"`
	RequiredTechCount      int16                  `json:"required_tech_count"`
	Civ                    int16                  `json:"civ"`
	FullTechMode           int16                  `json:"full_tech_mode"`
	LanguageDLLName        int32                  `json:"language_dll_name"`
	LanguageDLLDescription int32                  `json:"language_dll_description"`
	EffectID               int16                  `json:"effect_id"`
	Type                   int16                  `json:"type"`
	IconID                 int16                  `json:"icon_id"`
	LanguageDLLHelp        int32                  `json:"language_dll_help"`
	LanguageDLLTechTree    int32                  `json:"language_dll_tech_tree"`
	Name                   string                 `json:"name"`
	Repeatable             uint8                  `json:"repeatable"`
	ResearchLocations      []ResearchLocation     `json:"research_locations,omitempty"`
	Span                   Span                   `json:"span"`
}

type TechTree struct {
	AgeCount            int                  `json:"age_count"`
	BuildingCount       int                  `json:"building_count"`
	UnitCount           int                  `json:"unit_count"`
	ResearchCount       int                  `json:"research_count"`
	TotalUnitTechGroups int32                `json:"total_unit_tech_groups"`
	Ages                []TechTreeAge        `json:"ages,omitempty"`
	BuildingConnections []BuildingConnection `json:"building_connections,omitempty"`
	UnitConnections     []UnitConnection     `json:"unit_connections,omitempty"`
	ResearchConnections []ResearchConnection `json:"research_connections,omitempty"`
	Span                Span                 `json:"span"`
}

type TechTreeAge struct {
	Index              int            `json:"index"`
	ID                 int32          `json:"id"`
	Status             uint8          `json:"status"`
	Buildings          []int32        `json:"buildings,omitempty"`
	Units              []int32        `json:"units,omitempty"`
	Techs              []int32        `json:"techs,omitempty"`
	Common             TechTreeCommon `json:"common"`
	NumBuildingLevels  uint8          `json:"num_building_levels"`
	BuildingsPerZone   []uint8        `json:"buildings_per_zone,omitempty"`
	GroupLengthPerZone []uint8        `json:"group_length_per_zone,omitempty"`
	MaxAgeLength       uint8          `json:"max_age_length"`
	LineMode           int32          `json:"line_mode"`
	Span               Span           `json:"span"`
}

type TechTreeCommon struct {
	SlotsUsed    int32   `json:"slots_used"`
	UnitResearch []int32 `json:"unit_research,omitempty"`
	Mode         []int32 `json:"mode,omitempty"`
}

type Terrain struct {
	Index           int         `json:"index"`
	Name            DebugString `json:"name"`
	Name2           DebugString `json:"name_2"`
	OverlayMaskName DebugString `json:"overlay_mask_name"`
	Span            Span        `json:"span"`
}

type TerrainRestriction struct {
	Index       int                         `json:"index"`
	TerrainRows []TerrainRestrictionTerrain `json:"terrain_rows,omitempty"`
	Span        Span                        `json:"span"`
}

type TerrainRestrictionTerrain struct {
	Index              int     `json:"index"`
	Passability        float32 `json:"passability"`
	PassGraphicRawSize int     `json:"pass_graphic_raw_size"`
	Span               Span    `json:"span"`
	PassabilitySpan    Span    `json:"passability_span"`
	PassGraphicSpan    Span    `json:"pass_graphic_span"`
}

type TrainLocation struct {
	Index       int   `json:"index"`
	TrainTime   int16 `json:"train_time"`
	TrainUnitID int16 `json:"train_unit_id"`
	TrainButton uint8 `json:"train_button"`
	TrainHotkey int32 `json:"train_hotkey"`
	Span        Span  `json:"span"`
}

type TrainLocationPatch struct {
	Index       int    `json:"index"`
	TrainTime   *int16 `json:"train_time,omitempty"`
	TrainUnitID *int16 `json:"train_unit_id,omitempty"`
	TrainButton *uint8 `json:"train_button,omitempty"`
	TrainHotkey *int32 `json:"train_hotkey,omitempty"`
}

type TrainLocationRow struct {
	TrainTime   int16 `json:"train_time"`
	TrainUnitID int16 `json:"train_unit_id"`
	TrainButton uint8 `json:"train_button"`
	TrainHotkey int32 `json:"train_hotkey"`
}

type Type50FieldSpans struct {
	BaseArmor          Span `json:"base_armor"`
	ProjectileUnitID   Span `json:"projectile_unit_id"`
	MaxRange           Span `json:"max_range"`
	BlastWidth         Span `json:"blast_width"`
	AttackGraphic      Span `json:"attack_graphic"`
	BlastDamage        Span `json:"blast_damage"`
	MissileType        Span `json:"missile_type"`
	TargetingType      Span `json:"targeting_type"`
	MissileHitInfo     Span `json:"missile_hit_info"`
	MissileDieInfo     Span `json:"missile_die_info"`
	AreaEffectSpecials Span `json:"area_effect_specials"`
	BallisticsRatio    Span `json:"ballistics_ratio"`
	AttackCount        Span `json:"attack_count"`
	Attacks            Span `json:"attacks"`
	ArmourCount        Span `json:"armour_count"`
	Armours            Span `json:"armours"`
}

type Type50Summary struct {
	BaseArmor           int16            `json:"base_armor"`
	DefenseTerrainBonus int16            `json:"defense_terrain_bonus"`
	ProjectileUnitID    int16            `json:"projectile_unit_id"`
	MaxRange            float32          `json:"max_range"`
	BlastWidth          float32          `json:"blast_width"`
	AttackSpeed         float32          `json:"attack_speed"`
	BaseHitChance       int16            `json:"base_hit_chance"`
	BreakOffCombat      uint8            `json:"break_off_combat"`
	FrameDelay          int16            `json:"frame_delay"`
	WeaponOffsetX       float32          `json:"weapon_offset_x"`
	WeaponOffsetY       float32          `json:"weapon_offset_y"`
	WeaponOffsetZ       float32          `json:"weapon_offset_z"`
	BlastAttackLevel    uint8            `json:"blast_attack_level"`
	MinRange            float32          `json:"min_range"`
	MissedMissileSpread float32          `json:"missed_missile_spread"`
	AttackGraphic       int16            `json:"attack_graphic"`
	DisplayedArmor      int16            `json:"displayed_armor"`
	DisplayedAttack     int16            `json:"displayed_attack"`
	DisplayedRange      float32          `json:"displayed_range"`
	DisplayedReloadTime float32          `json:"displayed_reload_time"`
	BlastDamage         float32          `json:"blast_damage"`
	MissileType         uint8            `json:"missile_type"`
	TargetingType       uint8            `json:"targeting_type"`
	MissileHitInfo      uint8            `json:"missile_hit_info"`
	MissileDieInfo      uint8            `json:"missile_die_info"`
	AreaEffectSpecials  uint8            `json:"area_effect_specials"`
	BallisticsRatio     float32          `json:"ballistics_ratio"`
	Attacks             []WeaponInfo     `json:"attacks,omitempty"`
	Armours             []WeaponInfo     `json:"armours,omitempty"`
	FieldSpans          Type50FieldSpans `json:"field_spans,omitempty"`
}

type UnitAttribute struct {
	Index         int     `json:"index"`
	AttributeType uint16  `json:"attribute_type"`
	Amount        float32 `json:"amount"`
	Flag          uint8   `json:"flag"`
	Active        bool    `json:"active"`
	Span          Span    `json:"span"`
}

type UnitAttributePatch struct {
	Index         int      `json:"index"`
	AttributeType *uint16  `json:"attribute_type,omitempty"`
	Amount        *float32 `json:"amount,omitempty"`
	Flag          *uint8   `json:"flag,omitempty"`
}

type UnitConnection struct {
	Index            int            `json:"index"`
	ID               int32          `json:"id"`
	Status           uint8          `json:"status"`
	UpperBuilding    int32          `json:"upper_building"`
	Common           TechTreeCommon `json:"common"`
	VerticalLine     int32          `json:"vertical_line"`
	Units            []int32        `json:"units,omitempty"`
	LocationInAge    int32          `json:"location_in_age"`
	RequiredResearch int32          `json:"required_research"`
	LineMode         int32          `json:"line_mode"`
	EnablingResearch int32          `json:"enabling_research"`
	Span             Span           `json:"span"`
}

type UnitCreateRecipe struct {
	FromCivID                   int                   `json:"from_civ_id"`
	FromUnitID                  int                   `json:"from_unit_id"`
	CivIDs                      []int                 `json:"civ_ids,omitempty"`
	AllCivs                     *bool                 `json:"all_civs,omitempty"`
	HitPoints                   *int16                `json:"hit_points,omitempty"`
	Class                       *int16                `json:"class,omitempty"`
	LineOfSight                 *float32              `json:"line_of_sight,omitempty"`
	MovementType                *uint8                `json:"movement_type,omitempty"`
	StandingGraphic1            *int16                `json:"standing_graphic_1,omitempty"`
	StandingGraphic2            *int16                `json:"standing_graphic_2,omitempty"`
	DyingGraphic                *int16                `json:"dying_graphic,omitempty"`
	BloodUnitID                 *int16                `json:"blood_unit_id,omitempty"`
	IconID                      *int16                `json:"icon_id,omitempty"`
	Enabled                     *uint8                `json:"enabled,omitempty"`
	Attributes                  []UnitAttributePatch  `json:"attributes,omitempty"`
	DamageGraphics              []DamageGraphicPatch  `json:"damage_graphics,omitempty"`
	SetDamageGraphics           *[]DamageGraphicRow   `json:"set_damage_graphics,omitempty"`
	Type50ProjectileUnitID      *int16                `json:"type50_projectile_unit_id,omitempty"`
	Type50MaxRange              *float32              `json:"type50_max_range,omitempty"`
	Type50BlastWidth            *float32              `json:"type50_blast_width,omitempty"`
	Type50AttackGraphic         *int16                `json:"type50_attack_graphic,omitempty"`
	Type50BlastDamage           *float32              `json:"type50_blast_damage,omitempty"`
	Type50Attacks               []WeaponInfoPatch     `json:"type50_attacks,omitempty"`
	Type50Armours               []WeaponInfoPatch     `json:"type50_armours,omitempty"`
	SetType50Attacks            *[]WeaponInfoRow      `json:"set_type50_attacks,omitempty"`
	SetType50Armours            *[]WeaponInfoRow      `json:"set_type50_armours,omitempty"`
	TrainTime0                  *int16                `json:"train_time_0,omitempty"`
	TrainUnitID0                *int16                `json:"train_unit_id_0,omitempty"`
	TrainButtonID0              *uint8                `json:"train_button_id_0,omitempty"`
	TrainHotKeyID0              *int32                `json:"train_hotkey_id_0,omitempty"`
	CreatableButtonIconID       *int16                `json:"creatable_button_icon_id,omitempty"`
	CreatableButtonHotkeyAction *int16                `json:"creatable_button_hotkey_action,omitempty"`
	Costs                       []AttributeCostPatch  `json:"costs,omitempty"`
	TrainLocations              []TrainLocationPatch  `json:"train_locations,omitempty"`
	SetTrainLocations           *[]TrainLocationRow   `json:"set_train_locations,omitempty"`
	SetDropSites                *[]int16              `json:"set_drop_sites,omitempty"`
	LinkedBuildings             []LinkedBuildingPatch `json:"linked_buildings,omitempty"`
	Tasks                       []TaskPatch           `json:"tasks,omitempty"`
	SetTasks                    *[]TaskRow            `json:"set_tasks,omitempty"`
}

type UnitCreateReport struct {
	InputCompressedBytes  int                    `json:"input_compressed_bytes"`
	OutputCompressedBytes int                    `json:"output_compressed_bytes"`
	InputInflatedBytes    int                    `json:"input_inflated_bytes"`
	OutputInflatedBytes   int                    `json:"output_inflated_bytes"`
	InflatedLengthDelta   int                    `json:"inflated_length_delta"`
	FromCivID             int                    `json:"from_civ_id"`
	FromUnitID            int                    `json:"from_unit_id"`
	NewUnitID             int                    `json:"new_unit_id"`
	CivIDs                []int                  `json:"civ_ids"`
	CreatedCount          int                    `json:"created_count"`
	ChangedFields         []string               `json:"changed_fields"`
	Verified              bool                   `json:"verified"`
	Verification          aoe2.VerificationClaim `json:"verification"`
}

func CreateUnit(compressed []byte, recipe UnitCreateRecipe) ([]byte, UnitCreateReport, error)

type UnitFieldSpans struct {
	ID                 Span `json:"id"`
	Class              Span `json:"class"`
	HitPoints          Span `json:"hit_points"`
	LineOfSight        Span `json:"line_of_sight"`
	CollisionSizeX     Span `json:"collision_size_x"`
	CollisionSizeY     Span `json:"collision_size_y"`
	CollisionSizeZ     Span `json:"collision_size_z"`
	ClearanceSizeX     Span `json:"clearance_size_x"`
	ClearanceSizeY     Span `json:"clearance_size_y"`
	TerrainTableID     Span `json:"terrain_table_id"`
	MovementType       Span `json:"movement_type"`
	ObstructionTypeID  Span `json:"obstruction_type_id"`
	ObstructionClassID Span `json:"obstruction_class_id"`
	StandingGraphic1   Span `json:"standing_graphic_1"`
	StandingGraphic2   Span `json:"standing_graphic_2"`
	DyingGraphic       Span `json:"dying_graphic"`
	BloodUnitID        Span `json:"blood_unit_id"`
	IconID             Span `json:"icon_id"`
	Enabled            Span `json:"enabled"`
}

type UnitHeader struct {
	Index     int           `json:"index"`
	Exists    bool          `json:"exists"`
	TaskCount int           `json:"task_count"`
	Tasks     []TaskSummary `json:"tasks,omitempty"`
	Span      Span          `json:"span"`
}

type UnitListSpans struct {
	DamageGraphicCount Span `json:"damage_graphic_count"`
	DamageGraphics     Span `json:"damage_graphics"`
}

type UnitPatch struct {
	Class                       *int16                `json:"class,omitempty"`
	HitPoints                   *int16                `json:"hit_points,omitempty"`
	LineOfSight                 *float32              `json:"line_of_sight,omitempty"`
	MovementType                *uint8                `json:"movement_type,omitempty"`
	StandingGraphic1            *int16                `json:"standing_graphic_1,omitempty"`
	StandingGraphic2            *int16                `json:"standing_graphic_2,omitempty"`
	DyingGraphic                *int16                `json:"dying_graphic,omitempty"`
	BloodUnitID                 *int16                `json:"blood_unit_id,omitempty"`
	IconID                      *int16                `json:"icon_id,omitempty"`
	Enabled                     *uint8                `json:"enabled,omitempty"`
	Attributes                  []UnitAttributePatch  `json:"attributes,omitempty"`
	DamageGraphics              []DamageGraphicPatch  `json:"damage_graphics,omitempty"`
	SetDamageGraphics           *[]DamageGraphicRow   `json:"set_damage_graphics,omitempty"`
	Type50ProjectileUnitID      *int16                `json:"type50_projectile_unit_id,omitempty"`
	Type50MaxRange              *float32              `json:"type50_max_range,omitempty"`
	Type50BlastWidth            *float32              `json:"type50_blast_width,omitempty"`
	Type50AttackGraphic         *int16                `json:"type50_attack_graphic,omitempty"`
	Type50BlastDamage           *float32              `json:"type50_blast_damage,omitempty"`
	Type50Attacks               []WeaponInfoPatch     `json:"type50_attacks,omitempty"`
	Type50Armours               []WeaponInfoPatch     `json:"type50_armours,omitempty"`
	SetType50Attacks            *[]WeaponInfoRow      `json:"set_type50_attacks,omitempty"`
	SetType50Armours            *[]WeaponInfoRow      `json:"set_type50_armours,omitempty"`
	TrainTime0                  *int16                `json:"train_time_0,omitempty"`
	TrainUnitID0                *int16                `json:"train_unit_id_0,omitempty"`
	TrainButtonID0              *uint8                `json:"train_button_id_0,omitempty"`
	TrainHotKeyID0              *int32                `json:"train_hotkey_id_0,omitempty"`
	CreatableButtonIconID       *int16                `json:"creatable_button_icon_id,omitempty"`
	CreatableButtonHotkeyAction *int16                `json:"creatable_button_hotkey_action,omitempty"`
	Costs                       []AttributeCostPatch  `json:"costs,omitempty"`
	TrainLocations              []TrainLocationPatch  `json:"train_locations,omitempty"`
	SetTrainLocations           *[]TrainLocationRow   `json:"set_train_locations,omitempty"`
	SetDropSites                *[]int16              `json:"set_drop_sites,omitempty"`
	LinkedBuildings             []LinkedBuildingPatch `json:"linked_buildings,omitempty"`
	Tasks                       []TaskPatch           `json:"tasks,omitempty"`
	SetTasks                    *[]TaskRow            `json:"set_tasks,omitempty"`
}

func (patch UnitPatch) Empty() bool

type UnitPatchReport struct {
	InputCompressedBytes  int                    `json:"input_compressed_bytes"`
	OutputCompressedBytes int                    `json:"output_compressed_bytes"`
	InputInflatedBytes    int                    `json:"input_inflated_bytes"`
	OutputInflatedBytes   int                    `json:"output_inflated_bytes"`
	InflatedLengthDelta   int                    `json:"inflated_length_delta"`
	CivID                 int                    `json:"civ_id"`
	UnitID                int                    `json:"unit_id"`
	Before                UnitSummary            `json:"before"`
	After                 UnitSummary            `json:"after"`
	ChangedFields         []string               `json:"changed_fields"`
	Verified              bool                   `json:"verified"`
	Verification          aoe2.VerificationClaim `json:"verification"`
}

func PatchUnit(compressed []byte, civID, unitID int, patch UnitPatch) ([]byte, UnitPatchReport, error)

func PatchUnitFile(inputPath, outputPath string, civID, unitID int, patch UnitPatch) (UnitPatchReport, error)

func PatchUnitGraphicSlotAllCivs(compressed []byte, unitID int, slot string, graphicID int16) ([]byte, []UnitPatchReport, error)

type UnitRecipePatch struct {
	CivID                       int                   `json:"civ_id"`
	UnitID                      int                   `json:"unit_id"`
	HitPoints                   *int16                `json:"hit_points,omitempty"`
	Class                       *int16                `json:"class,omitempty"`
	LineOfSight                 *float32              `json:"line_of_sight,omitempty"`
	MovementType                *uint8                `json:"movement_type,omitempty"`
	StandingGraphic1            *int16                `json:"standing_graphic_1,omitempty"`
	StandingGraphic2            *int16                `json:"standing_graphic_2,omitempty"`
	DyingGraphic                *int16                `json:"dying_graphic,omitempty"`
	BloodUnitID                 *int16                `json:"blood_unit_id,omitempty"`
	IconID                      *int16                `json:"icon_id,omitempty"`
	Enabled                     *uint8                `json:"enabled,omitempty"`
	Attributes                  []UnitAttributePatch  `json:"attributes,omitempty"`
	DamageGraphics              []DamageGraphicPatch  `json:"damage_graphics,omitempty"`
	SetDamageGraphics           *[]DamageGraphicRow   `json:"set_damage_graphics,omitempty"`
	Type50ProjectileUnitID      *int16                `json:"type50_projectile_unit_id,omitempty"`
	Type50MaxRange              *float32              `json:"type50_max_range,omitempty"`
	Type50BlastWidth            *float32              `json:"type50_blast_width,omitempty"`
	Type50AttackGraphic         *int16                `json:"type50_attack_graphic,omitempty"`
	Type50BlastDamage           *float32              `json:"type50_blast_damage,omitempty"`
	Type50Attacks               []WeaponInfoPatch     `json:"type50_attacks,omitempty"`
	Type50Armours               []WeaponInfoPatch     `json:"type50_armours,omitempty"`
	SetType50Attacks            *[]WeaponInfoRow      `json:"set_type50_attacks,omitempty"`
	SetType50Armours            *[]WeaponInfoRow      `json:"set_type50_armours,omitempty"`
	TrainTime0                  *int16                `json:"train_time_0,omitempty"`
	TrainUnitID0                *int16                `json:"train_unit_id_0,omitempty"`
	TrainButtonID0              *uint8                `json:"train_button_id_0,omitempty"`
	TrainHotKeyID0              *int32                `json:"train_hotkey_id_0,omitempty"`
	CreatableButtonIconID       *int16                `json:"creatable_button_icon_id,omitempty"`
	CreatableButtonHotkeyAction *int16                `json:"creatable_button_hotkey_action,omitempty"`
	Costs                       []AttributeCostPatch  `json:"costs,omitempty"`
	TrainLocations              []TrainLocationPatch  `json:"train_locations,omitempty"`
	SetTrainLocations           *[]TrainLocationRow   `json:"set_train_locations,omitempty"`
	SetDropSites                *[]int16              `json:"set_drop_sites,omitempty"`
	LinkedBuildings             []LinkedBuildingPatch `json:"linked_buildings,omitempty"`
	Tasks                       []TaskPatch           `json:"tasks,omitempty"`
	SetTasks                    *[]TaskRow            `json:"set_tasks,omitempty"`
}

type UnitSummary struct {
	CivIndex           int               `json:"civ_index"`
	Index              int               `json:"index"`
	Present            bool              `json:"present"`
	Type               int               `json:"type"`
	ID                 int16             `json:"id"`
	Name               string            `json:"name"`
	Class              int16             `json:"class"`
	ClassName          string            `json:"class_name,omitempty"`
	HitPoints          int16             `json:"hit_points"`
	LineOfSight        float32           `json:"line_of_sight"`
	CollisionSizeX     float32           `json:"collision_size_x"`
	CollisionSizeY     float32           `json:"collision_size_y"`
	CollisionSizeZ     float32           `json:"collision_size_z"`
	ClearanceSizeX     float32           `json:"clearance_size_x"`
	ClearanceSizeY     float32           `json:"clearance_size_y"`
	TerrainTableID     int16             `json:"terrain_table_id"`
	MovementType       uint8             `json:"movement_type"`
	FlyMode            uint8             `json:"fly_mode"`
	ObstructionTypeID  uint8             `json:"obstruction_type_id"`
	ObstructionClassID uint8             `json:"obstruction_class_id"`
	StandingGraphic1   int16             `json:"standing_graphic_1"`
	StandingGraphic2   int16             `json:"standing_graphic_2"`
	DyingGraphic       int16             `json:"dying_graphic"`
	UndeadGraphic      int16             `json:"undead_graphic,omitempty"`
	UndeadFlag         uint8             `json:"undead_flag,omitempty"`
	BloodUnitID        int16             `json:"blood_unit_id"`
	TrainSound         int16             `json:"train_sound,omitempty"`
	DamageSound        int16             `json:"damage_sound,omitempty"`
	DeathSpawn         int16             `json:"death_spawn,omitempty"`
	GarrisonCapacity   uint8             `json:"garrison_capacity,omitempty"`
	IconID             int16             `json:"icon_id"`
	Enabled            uint8             `json:"enabled"`
	MoveSprite         int16             `json:"move_sprite,omitempty"`
	RunSprite          int16             `json:"run_sprite,omitempty"`
	TurnSpeed          float32           `json:"turn_speed,omitempty"`
	SizeClass          uint8             `json:"size_class,omitempty"`
	TrailingUnit       int16             `json:"trailing_unit,omitempty"`
	TrailingOptions    uint8             `json:"trailing_options,omitempty"`
	TrailingSpacing    float32           `json:"trailing_spacing,omitempty"`
	MoveAlgorithm      uint8             `json:"move_algorithm,omitempty"`
	TurnRadius         float32           `json:"turn_radius,omitempty"`
	TurnRadiusSpeed    float32           `json:"turn_radius_speed,omitempty"`
	MaxYawMoving       float32           `json:"maximum_yaw_per_second_moving,omitempty"`
	StationaryYawTime  float32           `json:"stationary_yaw_per_revolution_time,omitempty"`
	MaxYawStationary   float32           `json:"maximum_yaw_per_second_stationary,omitempty"`
	Attributes         []UnitAttribute   `json:"attributes,omitempty"`
	DamageGraphics     []DamageGraphic   `json:"damage_graphics,omitempty"`
	Action             *ActionSummary    `json:"action,omitempty"`
	Type50             *Type50Summary    `json:"type50,omitempty"`
	Creatable          *CreatableSummary `json:"creatable,omitempty"`
	Building           *BuildingSummary  `json:"building,omitempty"`
	FullFields         map[string]any    `json:"full_fields,omitempty"`
	FieldSpans         UnitFieldSpans    `json:"field_spans,omitempty"`
	ListSpans          UnitListSpans     `json:"list_spans,omitempty"`
	RecordStart        int               `json:"record_start"`
	RecordEnd          int               `json:"record_end"`
	RecordLength       int               `json:"record_length"`
}

func DecodeUnitFile(path string, civID, unitID int) (UnitSummary, map[string]any, error)
    DecodeUnitFile reads one unit record and projects the fields that the
    current DE parser can prove. It intentionally does not attach full maps to
    every unit in Index, which keeps normal DAT indexing compact.

func ParseUnitRecord(record []byte, civIndex, unitIndex int, version string, recordStart int) (UnitSummary, error)

type WeaponInfo struct {
	Index int   `json:"index"`
	Class int16 `json:"class"`
	Value int16 `json:"value"`
	Span  Span  `json:"span"`
}

type WeaponInfoPatch struct {
	Index int    `json:"index"`
	Class *int16 `json:"class,omitempty"`
	Value *int16 `json:"value,omitempty"`
}

type WeaponInfoRow struct {
	Class int16 `json:"class"`
	Value int16 `json:"value"`
}
```

## aoe2kit/pkg/diagnosticfixtures

```go
package diagnosticfixtures // import "aoe2kit/pkg/diagnosticfixtures"

Package diagnosticfixtures contains small, one-actor engine probes.

CONSTANTS

const (
	MapSize      = 120
	FireUnit     = 939
	FlowerUnit   = 1366
	BuildingUnit = 82
	ProbeTech    = 22 // Loom: valid stock tech, used only as a local-tech carrier.
)

TYPES

type Report struct {
	ScenarioPath string `json:"scenario_path"`
	XSPath       string `json:"xs_path"`
	TypesPath    string `json:"types_path"`
	RecipePath   string `json:"recipe_path"`
	ScenarioSHA  string `json:"scenario_sha256"`
	XSSHA        string `json:"xs_sha256"`
}

func BuildBarracksWall(dir string, timestamp int) (*Report, error)
    BuildBarracksWall creates a human-pathing A/B fixture. Each row belongs to a
    different allied player so type-level attribute changes cannot contaminate
    another row. Rows B-E have pre-placed and post-modification halves.

func BuildBarracksWallV2(dir string, timestamp int) (*Report, error)
    BuildBarracksWallV2 adds an unchanged spawned control to the wall and embeds
    a small XS telemetry probe for the engine-visible unit state.

func BuildHalo(dir string, timestamp int) (*Report, error)
    BuildHalo creates a fresh, one-active-player visual/retirement probe.

func BuildLocalTechnology(dir string, timestamp int) (*Report, error)
    BuildLocalTechnology creates two identical P1 buildings and applies one
    local technology to exactly one of them. The XS sidecar records both unit
    IDs and the engine-visible attributes selected by the run; it does not infer
    success.

func BuildRyanScratchAB(basePath, dir string, timestamp int) (*Report, error)
    BuildRyanScratchAB preserves the editor-authored P1 wall and adds two
    aligned spawned walls to the 1.59 scratch. P2 receives the four attribute
    changes before spawning; P3 is the unchanged spawned control.

func BuildRyanScratchAB2(basePath, dir string, timestamp int) (*Report, error)
    BuildRyanScratchAB2 creates a fresh five-wall clearance fixture from the
    editor-authored 1.59 scratch. It deliberately does not mutate or extend the
    existing RyanScratch or RyanScratch AB artifacts.
```

## aoe2kit/pkg/diagnostics

```go
package diagnostics // import "aoe2kit/pkg/diagnostics"


CONSTANTS

const SemanticsPackVersion = "dat-command-semantics-v4"
const SemanticsReadbackVersion = "dat-command-semantics-readback-v1"

FUNCTIONS

func WriteDATCommandSemanticsReadbackMarkdown(report *SemanticsReadbackReport, path string) error

TYPES

type ClassScopeEffectRow struct {
	EffectID      int      `json:"effect_id"`
	EffectName    string   `json:"effect_name"`
	Command       int      `json:"command"`
	Type          uint8    `json:"type"`
	TypeName      string   `json:"type_name"`
	Scope         string   `json:"scope,omitempty"`
	UnitID        int16    `json:"unit_id"`
	UnitClassID   int16    `json:"unit_class_id"`
	UnitClassName string   `json:"unit_class_name,omitempty"`
	AttributeID   int16    `json:"attribute_id"`
	AttributeName string   `json:"attribute_name,omitempty"`
	Amount        float32  `json:"amount"`
	Summary       string   `json:"summary"`
	Details       []string `json:"details,omitempty"`
	Warnings      []string `json:"warnings,omitempty"`
}

type ClassScopeEffectsReport struct {
	Version string                             `json:"version"`
	Feature string                             `json:"feature"`
	Status  string                             `json:"status"`
	Honesty []string                           `json:"honesty"`
	Matrix  datcodec.EffectCommandMatrixReport `json:"matrix"`
	Rows    []ClassScopeEffectRow              `json:"rows"`
}

func BuildClassScopeEffectsReport(datPath string) (ClassScopeEffectsReport, error)

type CommandClaim struct {
	Kind        string   `json:"kind,omitempty"`
	RawType     *uint8   `json:"raw_type,omitempty"`
	UnitID      *int16   `json:"unit_id,omitempty"`
	AttributeID *int16   `json:"attribute_id,omitempty"`
	TechID      *int     `json:"tech_id,omitempty"`
	Amount      *float32 `json:"amount,omitempty"`
}

type ExpectedLane struct {
	ID                  string         `json:"id"`
	Label               string         `json:"label"`
	Tier                string         `json:"tier"`
	DatEffectID         *int           `json:"dat_effect_id,omitempty"`
	DatEffectIDs        []int          `json:"dat_effect_ids,omitempty"`
	DatTechID           *int           `json:"dat_tech_id,omitempty"`
	DatTechIDs          []int          `json:"dat_tech_ids,omitempty"`
	ScenarioTriggerName string         `json:"scenario_trigger_name,omitempty"`
	Expected            []string       `json:"expected"`
	Evidence            []string       `json:"evidence"`
	Commands            []CommandClaim `json:"commands,omitempty"`
}

type ExpectedLedger struct {
	Version       string         `json:"version"`
	Status        string         `json:"status"`
	Honesty       []string       `json:"honesty"`
	Lanes         []ExpectedLane `json:"lanes"`
	PendingLanes  []ExpectedLane `json:"pending_lanes"`
	ReadbackSteps []string       `json:"readback_steps"`
}

type GeneratedFrom struct {
	DatPath         string `json:"dat_path"`
	BaseEffectCount int    `json:"base_effect_count"`
	BaseTechCount   int    `json:"base_tech_count"`
	TemplateTechID  int    `json:"template_tech_id"`
}

type LaneReadback struct {
	ID           string             `json:"id"`
	Label        string             `json:"label"`
	ExpectedTier string             `json:"expected_tier"`
	Status       string             `json:"status"`
	PromotedTier string             `json:"promoted_tier,omitempty"`
	Conclusion   string             `json:"conclusion"`
	Evidence     []ReadbackEvidence `json:"evidence"`
}

type LocalBuildingEffectRow struct {
	EffectID    int      `json:"effect_id"`
	EffectName  string   `json:"effect_name"`
	Command     int      `json:"command"`
	Type        uint8    `json:"type"`
	TypeName    string   `json:"type_name"`
	UnitID      int16    `json:"unit_id"`
	AttributeID int16    `json:"attribute_id"`
	Amount      float32  `json:"amount"`
	Summary     string   `json:"summary"`
	Details     []string `json:"details,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
}

type LocalBuildingEffectsReport struct {
	Version      string                             `json:"version"`
	Feature      string                             `json:"feature"`
	Status       string                             `json:"status"`
	Honesty      []string                           `json:"honesty"`
	CommandTypes []uint8                            `json:"command_types"`
	Matrix       datcodec.EffectCommandMatrixReport `json:"matrix"`
	Rows         []LocalBuildingEffectRow           `json:"rows"`
}

func BuildLocalBuildingEffectsReport(datPath string) (LocalBuildingEffectsReport, error)

type PackConstants struct {
	PlayerCivID      int `json:"player_civ_id"`
	VillagerUnitID   int `json:"villager_unit_id"`
	MilitiaUnitID    int `json:"militia_unit_id"`
	ManAtArmsUnitID  int `json:"man_at_arms_unit_id"`
	ScoutUnitID      int `json:"scout_unit_id"`
	SentinelUnitID   int `json:"sentinel_unit_id"`
	TownCenterUnitID int `json:"town_center_unit_id"`
	BarracksUnitID   int `json:"barracks_unit_id"`
	StableUnitID     int `json:"stable_unit_id"`
	CastleUnitID     int `json:"castle_unit_id"`
	HitPointsAttrID  int `json:"hit_points_attr_id"`
	FoodResourceID   int `json:"food_resource_id"`
	WoodResourceID   int `json:"wood_resource_id"`
	StoneResourceID  int `json:"stone_resource_id"`
	GoldResourceID   int `json:"gold_resource_id"`
}

type ReadbackEvidence struct {
	Source      string `json:"source"`
	Status      string `json:"status"`
	Observation string `json:"observation"`
}

type ReplayReadbackSummary struct {
	Duration               string `json:"duration,omitempty"`
	DurationMS             int    `json:"duration_ms,omitempty"`
	TriggerGraphOK         bool   `json:"trigger_graph_ok"`
	TriggerGraphError      string `json:"trigger_graph_error,omitempty"`
	ScenarioIdentityTier   string `json:"scenario_identity_tier,omitempty"`
	ScenarioIdentitySHA    string `json:"scenario_identity_sha256,omitempty"`
	DataSetStatus          string `json:"data_set_status,omitempty"`
	DataSetName            string `json:"data_set_name,omitempty"`
	DataSetSource          string `json:"data_set_source,omitempty"`
	DataSetConfidence      string `json:"data_set_confidence,omitempty"`
	DataSetError           string `json:"data_set_error,omitempty"`
	FinalResourceStockpile uint32 `json:"final_resource_stockpile,omitempty"`
	FinalObjectCount       int    `json:"final_object_count,omitempty"`
}

type SemanticsPack struct {
	Version        string          `json:"version"`
	Status         string          `json:"status"`
	GeneratedFrom  GeneratedFrom   `json:"generated_from"`
	Constants      PackConstants   `json:"constants"`
	DatRecipe      datcodec.Recipe `json:"dat_recipe"`
	ScenarioRecipe scenario.Recipe `json:"scenario_recipe"`
	Expected       ExpectedLedger  `json:"expected"`
}

func BuildDATCommandSemanticsPack(datPath string) (SemanticsPack, error)

func WriteDATCommandSemanticsPack(datPath, outDir string) (SemanticsPack, error)

func WriteDATCommandSemanticsPackWithOptions(datPath, outDir string, opts SemanticsPackOptions) (SemanticsPack, []string, error)

type SemanticsPackOptions struct {
	Feature string
}

type SemanticsReadbackInput struct {
	ExpectedPath string `json:"expected_path"`
	DatPath      string `json:"dat_path"`
	ScenarioPath string `json:"scenario_path"`
	ReplayPath   string `json:"replay_path"`
	XSDataPath   string `json:"xsdat_path,omitempty"`
}

type SemanticsReadbackOptions struct {
	ExpectedPath string
	DatPath      string
	ScenarioPath string
	ReplayPath   string
	XSDataPath   string
}

type SemanticsReadbackReport struct {
	Version      string                   `json:"version"`
	Verification string                   `json:"verification"`
	Status       string                   `json:"status"`
	Inputs       SemanticsReadbackInput   `json:"inputs"`
	Summary      SemanticsReadbackSummary `json:"summary"`
	Replay       ReplayReadbackSummary    `json:"replay"`
	Sidecar      SidecarReadbackSummary   `json:"sidecar,omitempty"`
	ParserGaps   []string                 `json:"parser_gaps,omitempty"`
	Lanes        []LaneReadback           `json:"lanes"`
}

func ReadDATCommandSemantics(opts SemanticsReadbackOptions) (*SemanticsReadbackReport, error)

type SemanticsReadbackSummary struct {
	Promoted       int `json:"promoted"`
	Failed         int `json:"failed"`
	Inconclusive   int `json:"inconclusive"`
	NotImplemented int `json:"not_implemented"`
}

type SidecarReadbackSummary struct {
	Path        string `json:"path,omitempty"`
	OK          bool   `json:"ok"`
	Schema      string `json:"schema,omitempty"`
	Magic       string `json:"magic,omitempty"`
	Version     int    `json:"version,omitempty"`
	Fixture     string `json:"fixture,omitempty"`
	Rows        int    `json:"rows,omitempty"`
	HasFooter   bool   `json:"has_footer,omitempty"`
	RowsWritten int    `json:"rows_written,omitempty"`
	Error       string `json:"error,omitempty"`
}
```

## aoe2kit/pkg/enginefacts

```go
package enginefacts // import "aoe2kit/pkg/enginefacts"


CONSTANTS

const (
	EngineVerified = "engine_verified"
	SourceVerified = "source_verified"
)

FUNCTIONS

func IsEngineVerified(id string) bool

TYPES

type Citation struct {
	FactID       string `json:"fact_id,omitempty"`
	Tier         string `json:"fact_tier,omitempty"`
	VerifiedDate string `json:"verified_date,omitempty"`
	FixtureRef   string `json:"fixture_ref,omitempty"`
}

func CitationFor(id string, includeProvisional bool) Citation

type Fact struct {
	ID           string   `json:"id"`
	Statement    string   `json:"statement"`
	Tier         string   `json:"tier"`
	VerifiedDate string   `json:"verified_date"`
	FixtureRef   string   `json:"fixture_ref"`
	Domains      []string `json:"domains,omitempty"`
	LintRule     *Rule    `json:"lint_rule,omitempty"`
}

type Ledger struct {
	SchemaVersion int    `json:"schema_version"`
	Generated     string `json:"generated"`
	Facts         []Fact `json:"facts"`
}

func LoadDefault() (Ledger, error)

func Parse(data []byte) (Ledger, error)

func (l Ledger) ByID(id string) (Fact, bool)

func (l Ledger) Filter(includeProvisional bool, domain string) []Fact

func (l Ledger) Sort()

func (l Ledger) Validate() error

type Rule struct {
	Target    string `json:"target,omitempty"`
	Predicate string `json:"predicate,omitempty"`
	Severity  string `json:"severity,omitempty"`
}
```

## aoe2kit/pkg/fx

```go
package fx // import "aoe2kit/pkg/fx"


CONSTANTS

const Version = "aoe2kit.fx.v1"

FUNCTIONS

func DescriptorJSON(name, preset string, grid Grid, overrides DescriptorOverrides) ([]byte, error)
func HasSingleBackslashAtlasFile(data []byte) bool
func NewDescriptor(name, preset string, grid Grid, overrides DescriptorOverrides) (map[string]any, error)

TYPES

type BindOptions struct {
	AtlasPath   string
	DDSPath     string
	Grid        Grid
	IntoCommon  string
	DatPath     string
	UnitID      int
	Slot        string
	FromGraphic int
	Name        string
	Preset      string
	ClearSLP    bool
	DryRun      bool
	Overrides   DescriptorOverrides
}

type BindReport struct {
	Version       string                 `json:"version"`
	OK            bool                   `json:"ok"`
	DryRun        bool                   `json:"dry_run,omitempty"`
	Verification  aoe2.VerificationClaim `json:"verification"`
	Name          string                 `json:"name"`
	Preset        string                 `json:"preset"`
	Grid          Grid                   `json:"grid"`
	Descriptor    string                 `json:"descriptor"`
	AtlasMeta     string                 `json:"atlas_meta"`
	AtlasImage    string                 `json:"atlas_image"`
	DatPath       string                 `json:"dat_path,omitempty"`
	UnitID        int                    `json:"unit_id,omitempty"`
	Slot          string                 `json:"slot,omitempty"`
	FromGraphic   int                    `json:"from_graphic,omitempty"`
	NewGraphicID  int                    `json:"new_graphic_id,omitempty"`
	UnitPatches   []UnitBindPatch        `json:"unit_patches,omitempty"`
	Checksums     map[string]string      `json:"checksums,omitempty"`
	DescriptorDoc map[string]any         `json:"descriptor_doc,omitempty"`
	Warnings      []string               `json:"warnings,omitempty"`
}

func Bind(options BindOptions) (BindReport, error)

type DDSInfo struct {
	Path       string `json:"path"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	FourCC     string `json:"fourcc"`
	DXT5       bool   `json:"dxt5"`
	HeaderSize uint32 `json:"header_size"`
}

func ReadDDS(path string) (DDSInfo, error)

type DescriptorLint struct {
	Name            string   `json:"name"`
	Path            string   `json:"path"`
	AtlasFile       string   `json:"atlas_file,omitempty"`
	AtlasPath       string   `json:"atlas_path,omitempty"`
	AtlasMetaPath   string   `json:"atlas_meta_path,omitempty"`
	ImageFirst      int      `json:"image_first"`
	ImageCount      int      `json:"image_count"`
	AtlasFrames     int      `json:"atlas_frames,omitempty"`
	ReferencedByDAT bool     `json:"referenced_by_dat,omitempty"`
	DDS             *DDSInfo `json:"dds,omitempty"`
	Errors          []string `json:"errors,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
}

type DescriptorOverrides struct {
	Type         *string
	Duration     *float64
	AlphaStart   *float64
	AlphaEnd     *float64
	Scale        *float64
	ScaleStart   *float64
	ScaleEnd     *float64
	Rotation     *float64
	StopMode     *string
	IsFire       *bool
	DisplayInFog *bool
}

type Grid struct {
	Rows   int `json:"rows"`
	Cols   int `json:"cols"`
	Frames int `json:"frames"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

func ParseFrameSize(raw string) (Grid, error)

func ParseGrid(raw string) (Grid, error)

type LintOptions struct {
	ModPath string
	DatPath string
}

type LintReport struct {
	Version      string                 `json:"version"`
	OK           bool                   `json:"ok"`
	Verification aoe2.VerificationClaim `json:"verification"`
	CommonRoot   string                 `json:"common_root"`
	DatPath      string                 `json:"dat_path,omitempty"`
	Descriptors  []DescriptorLint       `json:"descriptors"`
	Errors       []string               `json:"errors,omitempty"`
	Warnings     []string               `json:"warnings,omitempty"`
}

func Lint(options LintOptions) (LintReport, error)

type UnitBindPatch struct {
	CivID  int `json:"civ_id"`
	UnitID int `json:"unit_id"`
	Before int `json:"before"`
	After  int `json:"after"`
}
```

## aoe2kit/pkg/geom

```go
package geom // import "aoe2kit/pkg/geom"


CONSTANTS

const (
	MapMin          = 0
	MapMax          = 299
	DefaultMaxTiles = 40
)

VARIABLES

var Patterns = map[string]PatternSpec{
	"bouncing_dot":       {Name: "bouncing_dot", Generate: func() []Frame { return BouncingDot(12, 12, 36, 36, 32, 5) }},
	"checkerboard_sweep": {Name: "checkerboard_sweep", Generate: func() []Frame { return CheckerboardSweep(15, 15, 30, 30, 8) }},
	"collapsing_ring":    {Name: "collapsing_ring", Generate: func() []Frame { return CollapsingRing(30, 30, 14) }},
	"diamond_pulse":      {Name: "diamond_pulse", Generate: func() []Frame { return DiamondPulse(30, 30, 14) }},
	"dna_helix":          {Name: "dna_helix", Generate: func() []Frame { return DNAHelix(12, 30, 36, 8, 14, 24) }},
	"double_ring":        {Name: "double_ring", Generate: func() []Frame { return DoubleRing(30, 30, 14, 24) }},
	"expanding_ring":     {Name: "expanding_ring", Generate: func() []Frame { return ExpandingRing(30, 30, 14) }},
	"falling_comets":     {Name: "falling_comets", Generate: func() []Frame { return FallingComets(12, 12, 36, 36, 5, 24, 5) }},
	"line_sweep_x":       {Name: "line_sweep_x", Generate: func() []Frame { return LineSweep(12, 12, 36, 36, "x") }},
	"line_sweep_y":       {Name: "line_sweep_y", Generate: func() []Frame { return LineSweep(12, 12, 36, 36, "y") }},
	"lissajous":          {Name: "lissajous", Generate: func() []Frame { return Lissajous(30, 30, 18, 12, 3, 2, 32, 8) }},
	"orbiting_dots":      {Name: "orbiting_dots", Generate: func() []Frame { return OrbitingDots(30, 30, 16, 6, 24) }},
	"pinwheel":           {Name: "pinwheel", Generate: func() []Frame { return Pinwheel(30, 30, 18, 4, 24) }},
	"pulsing_star":       {Name: "pulsing_star", Generate: func() []Frame { return PulsingStar(30, 30, 18, 5, 24) }},
	"radar_sweep":        {Name: "radar_sweep", Generate: func() []Frame { return RadarSweep(30, 30, 18, 24) }},
	"ripple_grid":        {Name: "ripple_grid", Generate: func() []Frame { return RippleGrid(12, 12, 36, 36, 30, 30, 16) }},
	"rotating_square":    {Name: "rotating_square", Generate: func() []Frame { return RotatingPolygon(30, 30, 16, 4, 24) }},
	"rotating_triangle":  {Name: "rotating_triangle", Generate: func() []Frame { return RotatingPolygon(30, 30, 17, 3, 24) }},
	"sine_scroll":        {Name: "sine_scroll", Generate: func() []Frame { return SineScroll(12, 30, 36, 9, 18, 24) }},
	"snake":              {Name: "snake", Generate: func() []Frame { return Snake(12, 12, 36, 18, 32, 12) }},
	"spiral":             {Name: "spiral", Generate: func() []Frame { return Spiral(30, 30, 3, 18, 32, 6) }},
}

FUNCTIONS

func InBounds(t Tile) bool
func PatternNames() []string

TYPES

type Frame []Tile

func BouncingDot(x0, y0, w, h, steps, trail int) []Frame

func CheckerboardSweep(x0, y0, w, h, steps int) []Frame

func CollapsingRing(cx, cy, rMax int) []Frame

func DNAHelix(x0, y0, length int, amp, wavelength float64, steps int) []Frame

func DiamondPulse(cx, cy, rMax int) []Frame

func DoubleRing(cx, cy, rMax, steps int) []Frame

func ExpandingRing(cx, cy, rMax int) []Frame

func FallingComets(x0, y0, w, h, comets, steps, trail int) []Frame

func Generate(name string) ([]Frame, error)

func LineSweep(x0, y0, w, h int, axis string) []Frame

func Lissajous(cx, cy int, rx, ry float64, a, b, steps, tail int) []Frame

func OrbitingDots(cx, cy int, r float64, n, steps int) []Frame

func Pinwheel(cx, cy, r, arms, steps int) []Frame

func PulsingStar(cx, cy int, rMax float64, points, steps int) []Frame

func RadarSweep(cx, cy, r, steps int) []Frame

func RippleGrid(x0, y0, w, h, cx, cy, steps int) []Frame

func RotatingPolygon(cx, cy int, r float64, sides, steps int) []Frame

func SineScroll(x0, y0, w int, amp, wavelength float64, steps int) []Frame

func Snake(x0, y0, w, h, steps, length int) []Frame

func Spiral(cx, cy int, turns, r float64, steps, tail int) []Frame

type PatternFunc func() []Frame

type PatternSpec struct {
	Name     string
	Generate PatternFunc
}

type Tile struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Validation struct {
	Frames      int
	MaxTiles    int
	TotalTiles  int
	EmptyFrames int
	Errors      []string
}

func Validate(frames []Frame) Validation
```

## aoe2kit/pkg/geotrace

```go
package geotrace // import "aoe2kit/pkg/geotrace"


FUNCTIONS

func BuildDiamondLongitudeWrapTriggers(gridSize int, players []int, bandWidth int) []scenario.TriggerRecipe
func BuildEastWestWrapTriggers(gridSize int, players []int) []scenario.TriggerRecipe
func BuildRecipe(report Report) scenario.Recipe
func BuildSailDemoUnits(gridSize int) []scenario.UnitRecipe

TYPES

type BBox struct {
	MinLat float64 `json:"min_lat"`
	MinLon float64 `json:"min_lon"`
	MaxLat float64 `json:"max_lat"`
	MaxLon float64 `json:"max_lon"`
}

type Cell struct {
	X                    int             `json:"x"`
	Y                    int             `json:"y"`
	Lat                  float64         `json:"lat"`
	Lon                  float64         `json:"lon"`
	OutsideTrace         bool            `json:"outside_trace,omitempty"`
	Elevation            float64         `json:"elevation_m"`
	Land                 *bool           `json:"land,omitempty"`
	InlandWater          bool            `json:"inland_water,omitempty"`
	WaterbodyName        string          `json:"waterbody_name,omitempty"`
	LandcoverClass       int             `json:"landcover_class,omitempty"`
	LandcoverKey         string          `json:"landcover_key,omitempty"`
	LandcoverClassGLC    int             `json:"landcover_class_glc,omitempty"`
	LandcoverKeyGLC      string          `json:"landcover_key_glc,omitempty"`
	LandcoverFraction    float64         `json:"landcover_fraction,omitempty"`
	LandcoverHistogram   []LandcoverStat `json:"landcover_histogram,omitempty"`
	SpeciesUnitConsts    []int           `json:"species_unit_consts,omitempty"`
	DistanceToShoreTiles float64         `json:"distance_to_shore_tiles,omitempty"`
	ClimateZone          string          `json:"climate_zone,omitempty"`
	ElevationBand        string          `json:"elevation_band,omitempty"`
	Standing             string          `json:"standing"`
	TerrainID            int             `json:"terrain_id"`
	Road                 bool            `json:"road,omitempty"`
	Roadside             bool            `json:"roadside,omitempty"`
	WaterBuffer          bool            `json:"water_buffer,omitempty"`
	KeyWestCauseway      bool            `json:"key_west_causeway,omitempty"`
	Border               bool            `json:"border,omitempty"`
	ShoreCrossfadeBand   string          `json:"shore_crossfade_band,omitempty"`
	ShoreTerrainID       int             `json:"shore_terrain_id,omitempty"`
	TerrainConfidence    string          `json:"terrain_confidence,omitempty"`
	TerrainProportions   []TerrainWeight `json:"terrain_proportions,omitempty"`
	Reason               string          `json:"reason"`
}

type FeltMapping struct {
	Schema                      string                                        `json:"schema"`
	ConfidenceLadder            []string                                      `json:"confidence_ladder"`
	WaterByDistanceToShore      WaterDistanceMapping                          `json:"water_by_distance_to_shore"`
	LandByClimateAndBand        map[string]map[string]FeltTerrainChoice       `json:"-"`
	RawLandByClimateAndBand     map[string]json.RawMessage                    `json:"land_by_climate_and_band"`
	LandcoverClassMapping       *LandcoverClassMapping                        `json:"landcover_class_mapping,omitempty"`
	SpeciesForClusterScatter    map[string]map[string][]SpeciesRecommendation `json:"-"`
	RawSpeciesForClusterScatter map[string]json.RawMessage                    `json:"species_for_cluster_scatter"`
	LandcoverSpeciesByClass     map[string]map[string][]SpeciesRecommendation `json:"-"`
	RawLandcoverSpeciesByClass  map[string]json.RawMessage                    `json:"landcover_species_by_class"`
	SpeciesResolutions          []SpeciesResolution                           `json:"-"`
}

type FeltTerrainChoice struct {
	Proportions      []TerrainWeight                    `json:"proportions"`
	Confidence       string                             `json:"confidence"`
	Reason           string                             `json:"reason"`
	Note             string                             `json:"note,omitempty"`
	SpeciesByClimate map[string][]SpeciesRecommendation `json:"species_by_climate,omitempty"`
	WetlandCandidate *FeltTerrainChoice                 `json:"wetland_candidate,omitempty"`
}

type LandcoverClassMapping struct {
	ByClass map[string]FeltTerrainChoice `json:"by_class"`
}

type LandcoverSample struct {
	Valid     bool
	Class     int
	Key       string
	GLCClass  int
	GLCKey    string
	Fraction  float64
	Histogram []LandcoverStat
}

type LandcoverStat struct {
	Class  int     `json:"class"`
	Key    string  `json:"key"`
	Weight float64 `json:"weight"`
}

type Options struct {
	BBox                BBox
	GridSize            int
	Provider            string
	LandmaskProvider    string
	LandmaskPath        string
	InlandWaterProvider string
	InlandWaterPath     string
	LandcoverProvider   string
	LandcoverCacheDir   string
	LandcoverPath       string
	LandcoverDir        string
	ClimateZone         string
	FeltMappingPath     string
	Orient              string
	TargetWaterFraction float64
	SeamWaterColumns    int
	FloraDensity        float64
	LandFaunaDensity    float64
	FishDensity         float64
	ShoreFishDensity    float64
	OceanFishDensity    float64
	ShoreBandWidth      float64
	WaterBandWidth      float64
	RoadsProvider       string
	RoadsPath           string
	RoadsCacheDir       string
	RoadSimplifyTiles   float64
	AdminBorders        bool
	Client              *http.Client
}

type PlacementGuard struct {
	Policy                 string         `json:"policy"`
	Evidence               string         `json:"evidence"`
	UnbuildableTerrainIDs  []int          `json:"unbuildable_terrain_ids"`
	UnbuildableCells       map[int]int    `json:"unbuildable_cells,omitempty"`
	CandidateCellsFiltered int            `json:"candidate_cells_filtered,omitempty"`
	AllowedTerrainPrunes   map[string]int `json:"allowed_terrain_prunes,omitempty"`
}

type ProjectionSelfCheck struct {
	DueEastScreenDX  float64 `json:"due_east_screen_dx"`
	DueEastScreenDY  float64 `json:"due_east_screen_dy"`
	DueNorthScreenDX float64 `json:"due_north_screen_dx"`
	DueNorthScreenDY float64 `json:"due_north_screen_dy"`
}

type Report struct {
	BBox                  BBox                 `json:"bbox"`
	OverlayBBox           BBox                 `json:"overlay_bbox,omitempty"`
	GridSize              int                  `json:"grid_size"`
	Provider              string               `json:"provider"`
	LandmaskProvider      string               `json:"landmask_provider,omitempty"`
	InlandWaterProvider   string               `json:"inland_water_provider,omitempty"`
	LandcoverProvider     string               `json:"landcover_provider,omitempty"`
	FeltMappingSchema     string               `json:"felt_mapping_schema,omitempty"`
	ClimateZone           string               `json:"climate_zone,omitempty"`
	Orient                string               `json:"orient"`
	ApproxTileMeters      float64              `json:"approx_tile_meters"`
	CoastlineConfidence   string               `json:"coastline_confidence"`
	CoastlineReason       string               `json:"coastline_reason"`
	StandingLadder        []string             `json:"standing_ladder"`
	Warnings              []string             `json:"warnings,omitempty"`
	Cells                 []Cell               `json:"cells"`
	TerrainCounts         map[int]int          `json:"terrain_counts"`
	LandcoverCounts       map[string]int       `json:"landcover_counts,omitempty"`
	LandcoverGLCCounts    map[string]int       `json:"landcover_glc_counts,omitempty"`
	SpeciesResolutions    []SpeciesResolution  `json:"species_resolutions,omitempty"`
	LandCells             int                  `json:"land_cells"`
	WaterCells            int                  `json:"water_cells"`
	LandFraction          float64              `json:"land_fraction"`
	WaterFraction         float64              `json:"water_fraction"`
	TargetWaterFraction   float64              `json:"target_water_fraction,omitempty"`
	LandEmphasisApplied   bool                 `json:"land_emphasis_applied,omitempty"`
	FloraDensity          float64              `json:"flora_density,omitempty"`
	LandFaunaDensity      float64              `json:"land_fauna_density,omitempty"`
	FishDensity           float64              `json:"fish_density,omitempty"`
	ShoreFishDensity      float64              `json:"shore_fish_density,omitempty"`
	OceanFishDensity      float64              `json:"ocean_fish_density,omitempty"`
	ShoreBandWidth        float64              `json:"shore_band_width,omitempty"`
	WaterBandWidth        float64              `json:"water_band_width,omitempty"`
	ShoreCrossfadeCells   int                  `json:"shore_crossfade_cells,omitempty"`
	ShoreCrossfadeCounts  map[string]int       `json:"shore_crossfade_counts,omitempty"`
	ShoreTerrainCounts    map[int]int          `json:"shore_terrain_counts,omitempty"`
	UnbuildableTerrainIDs []int                `json:"unbuildable_terrain_ids,omitempty"`
	UnbuildableCells      int                  `json:"unbuildable_cells,omitempty"`
	PlacementGuard        *PlacementGuard      `json:"placement_guard,omitempty"`
	RoadsProvider         string               `json:"roads_provider,omitempty"`
	RoadWayCount          int                  `json:"road_way_count,omitempty"`
	RoadVertexCountRaw    int                  `json:"road_vertex_count_raw,omitempty"`
	RoadVertexCount       int                  `json:"road_vertex_count,omitempty"`
	RoadSimplifyTiles     float64              `json:"road_simplify_tiles,omitempty"`
	RoadCells             int                  `json:"road_cells,omitempty"`
	RoadsideCells         int                  `json:"roadside_cells,omitempty"`
	WaterBufferCells      int                  `json:"water_buffer_cells,omitempty"`
	KeyWestCausewayCells  int                  `json:"key_west_causeway_cells,omitempty"`
	BorderCells           int                  `json:"border_cells,omitempty"`
	BorderWayCount        int                  `json:"border_way_count,omitempty"`
	RoadCellsNorthOf31    int                  `json:"road_cells_north_of_31,omitempty"`
	BorderCellsNorthOf31  int                  `json:"border_cells_north_of_31,omitempty"`
	RoadRefs              []string             `json:"road_refs,omitempty"`
	LandExtentXTiles      int                  `json:"land_extent_x_tiles,omitempty"`
	LandExtentYTiles      int                  `json:"land_extent_y_tiles,omitempty"`
	LandExtentRatio       float64              `json:"land_extent_y_to_x_ratio,omitempty"`
	LandExtentWHRatio     float64              `json:"land_extent_w_to_h_ratio,omitempty"`
	ScreenWHRatio         float64              `json:"estimated_screen_w_to_h_ratio,omitempty"`
	ViewportTileWHRatio   float64              `json:"viewport_tile_w_to_h_ratio,omitempty"`
	ViewportScreenWHRatio float64              `json:"viewport_screen_w_to_h_ratio,omitempty"`
	PeninsulaAxisAngleDeg float64              `json:"peninsula_axis_angle_deg,omitempty"`
	ProjectionSelfCheck   *ProjectionSelfCheck `json:"projection_self_check,omitempty"`
	MinElevation          float64              `json:"min_elevation"`
	MaxElevation          float64              `json:"max_elevation"`
	MeanElevation         float64              `json:"mean_elevation"`
	Recipe                *scenario.Recipe     `json:"recipe,omitempty"`
}

func TraceDEM(opts Options) (Report, error)

type SpeciesRecommendation struct {
	UnitConst int    `json:"c"`
	Name      string `json:"s,omitempty"`
	Note      string `json:"note,omitempty"`
}

type SpeciesResolution struct {
	Name       string `json:"name"`
	UnitConst  int    `json:"unit_const"`
	Resolution string `json:"resolution"`
}

type TerrainWeight struct {
	TerrainID int     `json:"t"`
	Weight    float64 `json:"w"`
}

type WaterDistanceBucket struct {
	MaxDistanceTiles float64                    `json:"max_dist_tiles"`
	Confidence       string                     `json:"confidence"`
	ByClimate        map[string][]TerrainWeight `json:"by_climate"`
	Default          []TerrainWeight            `json:"default"`
}

type WaterDistanceMapping struct {
	Buckets []WaterDistanceBucket `json:"buckets"`
}
```

## aoe2kit/pkg/gfx

```go
package gfx // import "aoe2kit/pkg/gfx"


CONSTANTS

const Version = "aoe2kit.gfx.v1"

TYPES

type SLD struct {
	Path    string     `json:"path,omitempty"`
	Size    int        `json:"size"`
	Version uint16     `json:"version"`
	Frames  []SLDFrame `json:"frames"`

	// Has unexported fields.
}

func OpenSLD(path string) (*SLD, error)

func ParseSLD(data []byte) (*SLD, error)

type SLDExportOptions struct {
	OutDir string
	Limit  int
	Stats  bool
}

type SLDExportReport struct {
	Version      string                 `json:"version"`
	OK           bool                   `json:"ok"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Source       string                 `json:"source"`
	OutputDir    string                 `json:"output_dir"`
	Frames       int                    `json:"frames"`
	ManifestPath string                 `json:"manifest_path,omitempty"`
	ContactSheet string                 `json:"contact_sheet,omitempty"`
	Exported     []SLDExportedFrame     `json:"exported,omitempty"`
	Stats        []SLDFrameStats        `json:"stats,omitempty"`
	Warnings     []string               `json:"warnings,omitempty"`
}

func ExportSLD(path string, options SLDExportOptions) (SLDExportReport, error)

type SLDExportedFrame struct {
	FrameOrdinal int    `json:"frame_ordinal"`
	FrameIndex   uint16 `json:"frame_index"`
	Layer        string `json:"layer"`
	Path         string `json:"path"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	HotspotX     int    `json:"hotspot_x"`
	HotspotY     int    `json:"hotspot_y"`
	LayerOffsetX int    `json:"layer_offset_x"`
	LayerOffsetY int    `json:"layer_offset_y"`
	LayerWidth   int    `json:"layer_width"`
	LayerHeight  int    `json:"layer_height"`
	OpaquePixels int    `json:"opaque_pixels"`
}

type SLDFrame struct {
	Ordinal  int        `json:"ordinal"`
	Index    uint16     `json:"index"`
	Width    int        `json:"width"`
	Height   int        `json:"height"`
	HotspotX int        `json:"hotspot_x"`
	HotspotY int        `json:"hotspot_y"`
	Type     uint8      `json:"frame_type"`
	Unknown  uint8      `json:"unknown"`
	Start    int        `json:"start"`
	End      int        `json:"end"`
	Layers   []SLDLayer `json:"layers,omitempty"`
}

type SLDFrameBounds struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type SLDFrameStats struct {
	FrameOrdinal   int            `json:"frame_ordinal"`
	FrameIndex     uint16         `json:"frame_index"`
	CanvasWidth    int            `json:"canvas_width"`
	CanvasHeight   int            `json:"canvas_height"`
	Bounds         SLDFrameBounds `json:"bounds"`
	OpaquePixels   int            `json:"opaque_pixels"`
	CanvasCoverage float64        `json:"canvas_coverage"`
	MeanR          float64        `json:"mean_r"`
	MeanG          float64        `json:"mean_g"`
	MeanB          float64        `json:"mean_b"`
	MeanSaturation float64        `json:"mean_saturation"`
	HueBuckets     SLDHueBuckets  `json:"hue_buckets"`
	State          string         `json:"state"`
}

type SLDHueBuckets struct {
	Green  float64 `json:"green"`
	Yellow float64 `json:"yellow"`
	Red    float64 `json:"red"`
	Brown  float64 `json:"brown"`
	Grey   float64 `json:"grey"`
	White  float64 `json:"white"`
	Other  float64 `json:"other"`
}

type SLDLayer struct {
	Name          string `json:"name"`
	Compression   string `json:"compression,omitempty"`
	Start         int    `json:"start"`
	End           int    `json:"end"`
	ContentLength int    `json:"content_length"`
	PaddedLength  int    `json:"padded_length"`
	OffsetX1      int    `json:"offset_x1,omitempty"`
	OffsetY1      int    `json:"offset_y1,omitempty"`
	OffsetX2      int    `json:"offset_x2,omitempty"`
	OffsetY2      int    `json:"offset_y2,omitempty"`
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`
	Flag1         uint8  `json:"flag1,omitempty"`
	Unknown       uint8  `json:"unknown,omitempty"`
	CommandCount  int    `json:"command_count,omitempty"`
	DrawBlocks    int    `json:"draw_blocks,omitempty"`
}

type SLDManifest struct {
	Version      string                 `json:"version"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Source       string                 `json:"source"`
	FrameCount   int                    `json:"frame_count"`
	Frames       []SLDManifestFrame     `json:"frames"`
}

type SLDManifestFrame struct {
	Index        int    `json:"index"`
	FrameIndex   uint16 `json:"frame_index"`
	PNG          string `json:"png"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	AnchorX      int    `json:"anchor_x"`
	AnchorY      int    `json:"anchor_y"`
	LayerOffsetX int    `json:"layer_offset_x"`
	LayerOffsetY int    `json:"layer_offset_y"`
	LayerWidth   int    `json:"layer_width"`
	LayerHeight  int    `json:"layer_height"`
	OpaquePixels int    `json:"opaque_pixels"`
}
```

## aoe2kit/pkg/glyph

```go
package glyph // import "aoe2kit/pkg/glyph"

Package glyph converts TrueType outlines into the canonical Gaia-art cell
representation used by Kit's text and placement tooling.

TYPES

type Cell struct {
	X                 int     `json:"x"`
	Y                 int     `json:"y"`
	MaterialUnitConst int     `json:"material_unit_const"`
	Rotation          float64 `json:"rotation"`
}

type Glyph struct {
	Char   string `json:"char"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Cells  []Cell `json:"cells"`
}

type Library struct {
	Format      string  `json:"format"`
	Source      string  `json:"source"`
	Material    int     `json:"material_unit_const"`
	Rotation    float64 `json:"rotation"`
	CellScale   float64 `json:"cell_scale"`
	SampleSpace float64 `json:"sample_spacing"`
	Glyphs      []Glyph `json:"glyphs"`
}

func FromTTF(path string, opts Options) (Library, error)

type Options struct {
	Chars            []rune
	Material         int
	Rotation         float64
	CellScale        float64
	SampleSpacing    float64
	MaxCellsPerGlyph int
}

func DefaultOptions() Options
```

## aoe2kit/pkg/harness

```go
package harness // import "aoe2kit/pkg/harness"

Package harness generates unattended XS engine-fact probes.

CONSTANTS

const (
	BatchSafe      = "safe"
	BatchSixArm    = "six-arm"
	BatchTrigLong  = "trig-long"
	BatchAttrSweep = "attr-sweep"
	BatchLongV2    = "long-v2"
	BatchLongV3    = "long-v3"
	MapSize        = 120
)

TYPES

type Fact struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Setup       string `json:"setup"`
	Operation   string `json:"operation"`
	Observed    *int   `json:"observed,omitempty"`
	Expected    *int   `json:"expected,omitempty"`
	Risk        string `json:"risk"`
	Evidence    string `json:"evidence"`
}

func ExpandRemovalSweep(unitConsts []int) []Fact
    ExpandRemovalSweep expands one removal probe template into manifest entries.
    Expected is intentionally unset: the point of this sweep is to measure which
    units retire under HP zero, including units for which we have no prediction.

func Facts(batch string) []Fact

type Options struct {
	OutputDir           string
	ScenarioName        string
	Timestamp           int
	Batch               string
	XSCheckPath         string
	RequireXSCheck      bool
	AllowMissingXSCheck bool
}

type Report struct {
	OutputDir             string                   `json:"output_dir"`
	ScenarioPath          string                   `json:"scenario_path"`
	XSPath                string                   `json:"xs_path"`
	TypesPath             string                   `json:"types_path"`
	ManifestPath          string                   `json:"manifest_path"`
	RecipePath            string                   `json:"recipe_path"`
	ScenarioSHA           string                   `json:"scenario_sha256"`
	XSSHA                 string                   `json:"xs_sha256"`
	Batch                 string                   `json:"batch"`
	FactCount             int                      `json:"fact_count"`
	ExpectedRecords       int                      `json:"expected_records,omitempty"`
	EstimatedSidecarBytes int                      `json:"estimated_sidecar_bytes,omitempty"`
	EstimatedDurationS    int                      `json:"estimated_duration_s,omitempty"`
	SidecarByteCap        int                      `json:"sidecar_byte_cap,omitempty"`
	ExecutionCarrier      string                   `json:"execution_carrier"`
	XSParseCheck          string                   `json:"xs_parse_check"`
	XSParseChecker        string                   `json:"xs_parse_checker,omitempty"`
	XSParseOutput         string                   `json:"xs_parse_output,omitempty"`
	KitLintFindings       []xs.SourceFinding       `json:"kit_lint_findings,omitempty"`
	XSCheckFindings       []xs.ExternalFinding     `json:"xscheck_findings,omitempty"`
	Controversies         []xs.DifferentialFinding `json:"controversies,omitempty"`
	DifferentialStatus    string                   `json:"differential_status"`
}

func Build(opts Options) (*Report, error)
```

## aoe2kit/pkg/jsonout

```go
package jsonout // import "aoe2kit/pkg/jsonout"

Package jsonout is the single JSON boundary for user-facing command output.
It preserves ordinary encoding/json output and makes non-finite floats explicit
diagnostic strings instead of failing the entire command.

FUNCTIONS

func Encode(w io.Writer, value any) error
    Encode writes indented JSON to w. When non-finite floats are found, they
    are rendered as "NaN", "Infinity", or "-Infinity" and the object receives a
    nonfinite_count field. The command still succeeds because the value is data,
    not a transport error.

func Print(value any) error
    Print writes indented JSON to stdout.

func Sanitize(value any) (any, int)
    Sanitize converts non-finite float32/float64 values to explicit strings and
    returns the replacement count. It is exported so command/report tests can
    verify the boundary without invoking a subprocess.
```

## aoe2kit/pkg/kit

```go
package kit // import "aoe2kit/pkg/kit"


CONSTANTS

const ProfileMarkerName = "KIT_PROFILE.json"
const Version = "0.1.0"

FUNCTIONS

func CreditsJSON(root string) ([]byte, error)
func CreditsNoticeMarkdown(doc CreditsDocument) []byte
func KnownProfile(p PackProfile) bool
    KnownProfile reports whether a declared profile is one this kit understands.

func ManifestJSON() ([]byte, error)
func RecipeTemplatePayload(template RecipeTemplate) (any, error)
func RequiredDocs(profile PackProfile) []string
    RequiredDocs derives a non-public profile's required docs from its exclude
    rules. The public profile needs a tree; use RequiredDocsIn.

func RequiredDocsIn(root string, profile PackProfile) ([]string, error)
    RequiredDocs returns the artifacts a given profile promises to carry.
    It is the full list minus what the profile excludes, so the promise and
    the packing rules can never disagree. RequiredDocsIn is RequiredDocs for a
    concrete tree. The public profile is default-deny, so its required docs are
    the manifest docs that the tree's release/public-files.json lists.

func WriteArtifactLineage(path string, report ArtifactLineageReport) error

TYPES

type ArtifactLineageOptions struct {
	Tool    string
	Command []string
	Notes   []string
}

type ArtifactLineageReport struct {
	OK           bool            `json:"ok"`
	GeneratedAt  string          `json:"generated_at"`
	Verification string          `json:"verification"`
	Tool         string          `json:"tool,omitempty"`
	Command      []string        `json:"command,omitempty"`
	Inputs       []aoe2.FileInfo `json:"inputs"`
	Outputs      []aoe2.FileInfo `json:"outputs"`
	Notes        []string        `json:"notes,omitempty"`
	Warnings     []string        `json:"warnings,omitempty"`
}

func BuildArtifactLineage(inputs, outputs []string, opts ArtifactLineageOptions) (ArtifactLineageReport, error)

type Command struct {
	Name        string   `json:"name"`
	Summary     string   `json:"summary"`
	Subcommands []string `json:"subcommands,omitempty"`
}

type CreditEntry struct {
	ID              string   `json:"id"`
	Category        string   `json:"category"`
	Name            string   `json:"name"`
	Authors         []string `json:"authors,omitempty"`
	URL             string   `json:"url,omitempty"`
	License         string   `json:"license,omitempty"`
	Permission      string   `json:"permission,omitempty"`
	Gave            string   `json:"gave"`
	Role            string   `json:"role"`
	Relationship    string   `json:"relationship"`
	RungEarned      string   `json:"rung_earned"`
	ThanksStatus    string   `json:"thanks_status"`
	GiveBackStatus  string   `json:"give_back_status,omitempty"`
	CorrectionState string   `json:"correction_state"`
	Credit          string   `json:"credit"`
	Notes           []string `json:"notes,omitempty"`
	CreditClass     string   `json:"credit_class,omitempty"`
}

type CreditsDocument struct {
	Schema       string        `json:"schema"`
	Project      string        `json:"project"`
	License      string        `json:"license"`
	GeneratedBy  string        `json:"generated_by"`
	Dependencies []CreditEntry `json:"dependencies"`
	References   []CreditEntry `json:"references"`
}

func Credits(root string) (CreditsDocument, error)

type Inventory struct {
	Root     string                `json:"root"`
	Files    []aoe2.FileInfo       `json:"files"`
	ModDirs  []modpack.CheckReport `json:"mod_dirs"`
	Warnings []string              `json:"warnings,omitempty"`
}

func Scan(root string) (Inventory, error)

type Manifest struct {
	Name        string      `json:"name"`
	Version     string      `json:"version"`
	Neutrality  string      `json:"neutrality"`
	Runtime     RuntimeInfo `json:"runtime"`
	Docs        []string    `json:"docs"`
	Commands    []Command   `json:"commands"`
	Domains     []string    `json:"domains"`
	Limitations []string    `json:"limitations"`
}

func CurrentManifest() Manifest

type PackOptions struct {
	Profile        PackProfile
	Exclude        []string // additional glob patterns, matched against the relative path
	SHASidecar     bool     // write <output>.sha256 next to the archive
	KeepBinary     *bool    // override the profile's binary decision
	UpdateManifest bool     // refresh KIT_MANIFEST.json before verifying
}

type PackProfile string
    PackProfile names an audience, because that is what actually decides what
    belongs in an archive.

        full     everything tracked, plus the prebuilt binary — our own archive
        handoff  source, data, and root-level documents for someone who has Go
        sandbox  handoff plus the binary, for a container that can run but not compile
        public   default-deny: exactly the files listed in release/public-files.json

const (
	ProfileFull    PackProfile = "full"
	ProfileHandoff PackProfile = "handoff"
	ProfileSandbox PackProfile = "sandbox"
	ProfilePublic  PackProfile = "public"
)
type PackReport struct {
	Root            string                 `json:"root"`
	Output          string                 `json:"output"`
	Profile         string                 `json:"profile,omitempty"`
	SHASidecar      string                 `json:"sha256_sidecar,omitempty"`
	ManifestUpdated bool                   `json:"manifest_updated,omitempty"`
	FileCount       int                    `json:"file_count"`
	Bytes           int64                  `json:"bytes"`
	SHA256          string                 `json:"sha256"`
	Verified        bool                   `json:"verified"`
	Verification    aoe2.VerificationClaim `json:"verification"`
	VerifyError     string                 `json:"verify_error,omitempty"`
}

func Pack(root, outputPath string) (PackReport, error)
    Pack keeps the original signature and behavior: a full archive.

func PackWithOptions(root, outputPath string, opts PackOptions) (PackReport, error)

type PortableCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

type PortableCheckOptions struct {
	Profile PackProfile
}

type PortableReport struct {
	Root     string          `json:"root"`
	Profile  string          `json:"profile,omitempty"`
	OK       bool            `json:"ok"`
	Checks   []PortableCheck `json:"checks"`
	Warnings []string        `json:"warnings,omitempty"`
	Errors   []string        `json:"errors,omitempty"`
}

func PortableCheckRoot(root string) (PortableReport, error)

func PortableCheckRootWithOptions(root string, opts PortableCheckOptions) (PortableReport, error)

type ProfileMarker struct {
	Profile    string `json:"profile"`
	KitVersion string `json:"kit_version"`
	PackedAt   string `json:"packed_at"`
	// Generation counts how many times this kit has been packed and repacked.
	// A copy three hands down can say so without any central registry.
	Generation int    `json:"generation"`
	ParentSHA  string `json:"parent_sha256,omitempty"`
}

func ReadProfileMarker(root string) (ProfileMarker, error)
    ReadProfileMarker returns the archive's declared profile. A tree with no
    marker is a full kit: that is what every existing checkout is. An unreadable
    or unrecognized marker is reported rather than quietly treated as full — on
    a redistribution gate, "I do not know what this archive claims to be" must
    not look like "this archive is fine".

type ProjectDiffReport struct {
	Before       string              `json:"before"`
	After        string              `json:"after"`
	Same         bool                `json:"same"`
	Verification string              `json:"verification"`
	Summary      ProjectDiffSummary  `json:"summary"`
	Changes      []ProjectFileChange `json:"changes,omitempty"`
}

func DiffProjectSnapshots(beforePath, afterPath string, limit int) (ProjectDiffReport, error)

type ProjectDiffSummary struct {
	Added    int `json:"added"`
	Removed  int `json:"removed"`
	Modified int `json:"modified"`
	Shown    int `json:"shown"`
	Limit    int `json:"limit,omitempty"`
}

type ProjectFile struct {
	Path      string   `json:"path"`
	Kind      string   `json:"kind"`
	Role      string   `json:"role"`
	SizeBytes int64    `json:"size_bytes"`
	SHA256    string   `json:"sha256,omitempty"`
	Reasons   []string `json:"reasons,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
}

type ProjectFileChange struct {
	Path   string       `json:"path"`
	Change string       `json:"change"`
	Before *ProjectFile `json:"before,omitempty"`
	After  *ProjectFile `json:"after,omitempty"`
	Fields []string     `json:"fields,omitempty"`
}

type ProjectInspectOptions struct {
	Limit int
}

type ProjectInspectReport struct {
	Root            string         `json:"root"`
	Verification    string         `json:"verification"`
	FilesScanned    int            `json:"files_scanned"`
	Shown           int            `json:"shown"`
	RoleCounts      map[string]int `json:"role_counts"`
	KindCounts      map[string]int `json:"kind_counts"`
	Files           []ProjectFile  `json:"files"`
	Warnings        []string       `json:"warnings,omitempty"`
	Recommendations []string       `json:"recommendations,omitempty"`
}

func InspectProject(root string, opts ProjectInspectOptions) (ProjectInspectReport, error)

type ProjectSnapshotReport struct {
	Root         string               `json:"root"`
	Output       string               `json:"output,omitempty"`
	OK           bool                 `json:"ok"`
	Verification string               `json:"verification"`
	Inspect      ProjectInspectReport `json:"inspect"`
}

func LoadProjectSnapshot(path string) (ProjectSnapshotReport, error)

func WriteProjectSnapshot(root, output string) (ProjectSnapshotReport, error)

type RecipeTemplate struct {
	Name           string           `json:"name"`
	Domain         string           `json:"domain"`
	Summary        string           `json:"summary"`
	AppliesTo      string           `json:"applies_to"`
	Command        string           `json:"command"`
	Verification   string           `json:"verification"`
	Notes          []string         `json:"notes,omitempty"`
	ScenarioRecipe *scenario.Recipe `json:"scenario_recipe,omitempty"`
	DatRecipe      *datcodec.Recipe `json:"dat_recipe,omitempty"`
}

func RecipeTemplateByName(name string) (RecipeTemplate, bool)

func RecipeTemplates() []RecipeTemplate

type RecipeTemplateExportFile struct {
	Name        string `json:"name"`
	Domain      string `json:"domain"`
	Path        string `json:"path"`
	Bytes       int    `json:"bytes"`
	Overwritten bool   `json:"overwritten,omitempty"`
}

type RecipeTemplateExportReport struct {
	Verification string                     `json:"verification"`
	OutputDir    string                     `json:"output_dir"`
	Domain       string                     `json:"domain,omitempty"`
	Count        int                        `json:"count"`
	Files        []RecipeTemplateExportFile `json:"files"`
}

func ExportRecipeTemplates(outDir, domain string, force bool) (RecipeTemplateExportReport, error)

type RecipeTemplateListReport struct {
	Verification string                  `json:"verification"`
	Domain       string                  `json:"domain,omitempty"`
	Count        int                     `json:"count"`
	Templates    []RecipeTemplateSummary `json:"templates"`
}

func RecipeTemplateSummaries(domain string) (RecipeTemplateListReport, error)

type RecipeTemplateSummary struct {
	Name         string   `json:"name"`
	Domain       string   `json:"domain"`
	Summary      string   `json:"summary"`
	AppliesTo    string   `json:"applies_to"`
	Command      string   `json:"command"`
	Verification string   `json:"verification"`
	Notes        []string `json:"notes,omitempty"`
}

type RuntimeInfo struct {
	DurableLanguage string `json:"durable_language"`
	PythonRequired  bool   `json:"python_required"`
	NetworkRequired bool   `json:"network_required"`
}

type VerifyReport struct {
	Root          string                 `json:"root"`
	Version       string                 `json:"version"`
	Profile       string                 `json:"profile,omitempty"`
	Generation    int                    `json:"generation,omitempty"`
	SourceBuildOK bool                   `json:"source_build_ok"`
	DocsOK        bool                   `json:"docs_ok"`
	ManifestOK    bool                   `json:"manifest_ok"`
	InventoryOK   bool                   `json:"inventory_ok"`
	ArtifactsOK   bool                   `json:"artifacts_ok"`
	GoAvailable   bool                   `json:"go_available"`
	Verification  aoe2.VerificationClaim `json:"verification"`
	Warnings      []string               `json:"warnings,omitempty"`
	Errors        []string               `json:"errors,omitempty"`
}

func Verify(root string, runGoChecks bool) VerifyReport

func (r VerifyReport) OK() bool
```

## aoe2kit/pkg/modpack

```go
package modpack // import "aoe2kit/pkg/modpack"


TYPES

type ActivationReport struct {
	Checked        bool             `json:"checked"`
	StatusPath     string           `json:"status_path,omitempty"`
	ModTitle       string           `json:"mod_title,omitempty"`
	DataMod        bool             `json:"data_mod"`
	Matched        []ModStatusEntry `json:"matched,omitempty"`
	Enabled        *bool            `json:"enabled,omitempty"`
	Published      *bool            `json:"published,omitempty"`
	ActiveDataMods []string         `json:"active_data_mods,omitempty"`
	ActivationOK   bool             `json:"activation_ok"`
	Warnings       []string         `json:"warnings,omitempty"`
	Errors         []string         `json:"errors,omitempty"`
}

func CheckActivation(modPath, title string, dataMod bool, statusPath string) ActivationReport

type CheckOptions struct {
	StatusPath string
}

type CheckReport struct {
	Path            string                 `json:"path"`
	InfoJSON        bool                   `json:"info_json"`
	Info            map[string]any         `json:"info,omitempty"`
	Title           string                 `json:"title,omitempty"`
	Verification    aoe2.VerificationClaim `json:"verification"`
	ResourcesCommon bool                   `json:"resources_common"`
	DataFiles       []string               `json:"data_files,omitempty"`
	StringFiles     []string               `json:"string_files,omitempty"`
	AIFiles         []string               `json:"ai_files,omitempty"`
	XSFiles         []string               `json:"xs_files,omitempty"`
	DRSFiles        []string               `json:"drs_files,omitempty"`
	ScenarioFiles   []string               `json:"scenario_files,omitempty"`
	OtherKnownFiles []string               `json:"other_known_files,omitempty"`
	Activation      *ActivationReport      `json:"activation,omitempty"`
	Warnings        []string               `json:"warnings,omitempty"`
	Errors          []string               `json:"errors,omitempty"`
}

func Check(path string) (CheckReport, error)

func CheckWithOptions(path string, opts CheckOptions) (CheckReport, error)

func (r CheckReport) OK() bool

type ModStatusEntry struct {
	Title     string         `json:"title,omitempty"`
	ID        string         `json:"id,omitempty"`
	Path      string         `json:"path,omitempty"`
	Enabled   *bool          `json:"enabled,omitempty"`
	Published *bool          `json:"published,omitempty"`
	Type      string         `json:"type,omitempty"`
	RawKeys   map[string]any `json:"raw_keys,omitempty"`
}
```

## aoe2kit/pkg/registry

```go
package registry // import "aoe2kit/pkg/registry"


FUNCTIONS

func Keys(registry Registry) []string
func Save(path string, registry Registry) error

TYPES

type Entry struct {
	Name         string `json:"name"`
	Family       string `json:"family,omitempty"`
	Description  string `json:"description,omitempty"`
	Tier         string `json:"tier,omitempty"`
	Fingerprint  string `json:"fingerprint,omitempty"`
	TriggerCount int    `json:"trigger_count,omitempty"`
	Notes        string `json:"notes,omitempty"`
}

type Registry map[string]Entry

func Load(path string) (Registry, error)

func Register(path, hash string, entry Entry) (Registry, error)
```

## aoe2kit/pkg/replay

```go
package replay // import "aoe2kit/pkg/replay"


CONSTANTS

const (
	DefaultEffectiveAvailabilityOffset = 879618
	DefaultEffectiveAvailabilityCount  = 864

	ObservedVanillaEffectiveTemplateName   = "observed_de_v68_vanilla_effective_template_2026_07_21"
	ObservedVanillaEffectiveTemplateSHA256 = "cba7550a169f61ba9bafd751df2fec8a7b02fabef5acaa4245e07f2fed86b9d0"
)
const PlaytestReportVersion = "aoe2kit.replay.playtest.v1"
const PromisoryRootEnv = "AOE2KIT_PROMISORY_ROOT"
const SummaryVersion = "aoe2kit.replay.summary.v1"

FUNCTIONS

func AnnotateEventObjects(events []ReplayEvent, index *ObjectIndexReport)
func AnnotateFeedbackPhases(events []ReplayEvent)
func AnnotateRegions(events []ReplayEvent, context *Context)
func ChannelName(channel int) string
func CivDisplayName(civID int) string
func ExtractActionStreamEvents(body []byte, opts EventOptions) ([]ReplayEvent, map[int]string, EventCounts, []string)
func ExtractPrintableStrings(data []byte, minLen int) []string
func FormatTime(ms int) string
func HexSample(data []byte, limit int) string
func ParseClockMS(value string) (int, error)
func ReadRecordBytes(path string) ([]byte, error)
func SortReplayEvents(events []ReplayEvent)
func TauntText(number int) string
func TrackMemory(out io.Writer) func()
    TrackMemory samples Go allocation counters. Linux VmHWM is reported
    separately: runtime.Sys is not RSS, and sampling can miss brief heap peaks.

func UnitDisplayName(unitID int) string
func WriteHealthText(out io.Writer, r *HealthReport)
func WriteSyncJSON(out io.Writer, path string, opts SyncOptions) error
    WriteSyncJSON streams all arrays from cache. The library's slice-returning
    API remains bounded and fails explicitly if a caller asks for too much.


TYPES

type AIManifestMention struct {
	Space  string `json:"space"`
	Offset int    `json:"offset"`
	End    int    `json:"end"`
	Region string `json:"region,omitempty"`
	Status string `json:"status,omitempty"`
}

type AIManifestModule struct {
	Name         string `json:"name"`
	Count        int    `json:"count"`
	ResolvedPath string `json:"resolved_path,omitempty"`
	Resolved     bool   `json:"resolved"`
}

type AIManifestOptions struct {
	PromisoryRoot string
}

type AIManifestReference struct {
	ModulePath   string `json:"module_path"`
	ModuleName   string `json:"module_name"`
	Directive    string `json:"directive"`
	Space        string `json:"space"`
	Start        int    `json:"start"`
	End          int    `json:"end"`
	Region       string `json:"region"`
	SourceRegion string `json:"source_region,omitempty"`
	ResolvedPath string `json:"resolved_path,omitempty"`
	Resolved     bool   `json:"resolved"`
}

func (r AIManifestReference) String() string

type AIManifestReport struct {
	Path            string                 `json:"path,omitempty"`
	Method          string                 `json:"method"`
	Verification    aoe2.VerificationClaim `json:"verification"`
	Summary         AIManifestSummary      `json:"summary"`
	PromisoryRoot   string                 `json:"promisory_root,omitempty"`
	PromisoryRootOK bool                   `json:"promisory_root_ok"`
	PromiDEMentions []AIManifestMention    `json:"promide_mentions,omitempty"`
	References      []AIManifestReference  `json:"references,omitempty"`
	XSIncludes      []AIManifestXSInclude  `json:"xs_includes,omitempty"`
	UniqueModules   []AIManifestModule     `json:"unique_modules,omitempty"`
	Warnings        []string               `json:"warnings,omitempty"`
}

func BuildAIManifest(path string, opts AIManifestOptions) (*AIManifestReport, error)

type AIManifestSummary struct {
	PromiDEMentions     int `json:"promide_mentions"`
	LoadDirectives      int `json:"load_directives"`
	XSIncludes          int `json:"xs_includes"`
	UniqueModules       int `json:"unique_modules"`
	ResolvedModules     int `json:"resolved_modules"`
	MissingModules      int `json:"missing_modules"`
	HeaderOpaqueBytes   int `json:"header_opaque_bytes"`
	HeaderDecodedBytes  int `json:"header_decoded_bytes"`
	InflatedHeaderBytes int `json:"inflated_header_bytes"`
}

type AIManifestXSInclude struct {
	IncludePath  string `json:"include_path"`
	IncludeName  string `json:"include_name"`
	Directive    string `json:"directive"`
	Space        string `json:"space"`
	Start        int    `json:"start"`
	End          int    `json:"end"`
	Region       string `json:"region"`
	SourceRegion string `json:"source_region,omitempty"`
}

type APMWindow struct {
	StartMS int     `json:"start_ms"`
	EndMS   int     `json:"end_ms"`
	Start   string  `json:"start"`
	End     string  `json:"end"`
	Actions int     `json:"actions"`
	APM     float64 `json:"apm"`
}

type Action255Postgame struct {
	SourceOffset int    `json:"source_offset"`
	TimeMS       int    `json:"time_ms,omitempty"`
	Time         string `json:"time,omitempty"`
	PayloadBytes int    `json:"payload_bytes"`
	Sequence     int    `json:"sequence,omitempty"`
	Confidence   string `json:"confidence"`
}

type AttributeMirrorCandidate struct {
	Attribute string `json:"attribute"`
	Value     int    `json:"value"`
	WordIndex int    `json:"word_index"`
}

type CacheManifest struct {
	Version      int      `json:"version"`
	SHA256       string   `json:"sha256"`
	HeaderLength int64    `json:"header_length"`
	HeaderBytes  int64    `json:"header_bytes"`
	RecordBytes  int64    `json:"record_bytes"`
	OpsBytes     int64    `json:"ops_bytes"`
	DurationMS   int      `json:"duration_ms"`
	Warnings     []string `json:"warnings,omitempty"`
}

type CachedOp struct {
	ID           uint32     `json:"op"`
	Offset       int        `json:"offset"`
	TimeMS       int        `json:"time_ms"`
	ActionID     int        `json:"action_id,omitempty"`
	Sequence     int        `json:"sequence,omitempty"`
	Payload      []byte     `json:"payload,omitempty"`
	Sync         *SyncEvent `json:"sync,omitempty"`
	Terminal     bool       `json:"terminal,omitempty"`
	SkippedBytes int        `json:"skipped_bytes,omitempty"`
}

type CameraEvent struct {
	TimeMS       int     `json:"time_ms"`
	Time         string  `json:"time"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	TailU32      uint32  `json:"tail_u32"`
	SourceOffset int     `json:"source_offset"`
}

type CameraObservation struct {
	TimeMS int     `json:"time_ms"`
	Time   string  `json:"time"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Region string  `json:"region,omitempty"`
}

type CameraOptions struct {
	// Limit caps emitted event rows (0 = all). Summaries always cover every event.
	Limit int
	// Tail filters events/streams to a single tail_u32 value when >= 0.
	Tail int
}

type CameraReport struct {
	Path         string         `json:"path,omitempty"`
	Method       string         `json:"method"`
	Verification string         `json:"verification"`
	Summary      CameraSummary  `json:"summary"`
	Streams      []CameraStream `json:"streams,omitempty"`
	Events       []CameraEvent  `json:"events,omitempty"`
	Warnings     []string       `json:"warnings,omitempty"`
}

func BuildCamera(path string, opts CameraOptions) (*CameraReport, error)

type CameraStream struct {
	TailU32            uint32  `json:"tail_u32"`
	TailHex            string  `json:"tail_hex"`
	PlayerNumberIfTail int     `json:"player_number_if_tail_is_player,omitempty"`
	PlayerNameIfTail   string  `json:"player_name_if_tail_is_player,omitempty"`
	MappingConfidence  string  `json:"mapping_confidence"`
	Events             int     `json:"events"`
	FirstTimeMS        int     `json:"first_time_ms"`
	FirstTime          string  `json:"first_time"`
	LastTimeMS         int     `json:"last_time_ms"`
	LastTime           string  `json:"last_time"`
	MinX               float64 `json:"min_x"`
	MaxX               float64 `json:"max_x"`
	MinY               float64 `json:"min_y"`
	MaxY               float64 `json:"max_y"`
	DistanceTraveled   float64 `json:"distance_traveled"`
	MeanStepDistance   float64 `json:"mean_step_distance"`
	MaxStepDistance    float64 `json:"max_step_distance"`
	LargeJumps         int     `json:"large_jumps_over_20"`
}

type CameraSummary struct {
	Events           int    `json:"events"`
	EmittedEvents    int    `json:"emitted_events"`
	DurationMS       int    `json:"duration_ms"`
	Duration         string `json:"duration"`
	DistinctTails    int    `json:"distinct_tail_values"`
	TailsMatchRoster bool   `json:"tail_values_coincide_with_player_numbers"`
	TailMappingNote  string `json:"tail_mapping_note"`
}

type CarrierAttempt struct {
	Index          int             `json:"index"`
	TimerS         int             `json:"timer_s"`
	WindowStartMS  int             `json:"window_start_ms"`
	WindowEndMS    int             `json:"window_end_ms,omitempty"`
	Label          string          `json:"label,omitempty"`
	XSOpenFile     string          `json:"xs_open_file"`
	ExpectedValues map[string]int  `json:"expected_values"`
	Observed       []CarrierSample `json:"observed,omitempty"`
	Verdict        string          `json:"verdict"`
	MatchedKey     string          `json:"matched_key,omitempty"`
	MatchedValue   int             `json:"matched_value,omitempty"`
	Confidence     string          `json:"confidence"`
	Note           string          `json:"note,omitempty"`
}

type CarrierOptions struct {
	LedgerPath string
}

type CarrierReport struct {
	Path         string           `json:"path,omitempty"`
	LedgerPath   string           `json:"ledger_path,omitempty"`
	Method       string           `json:"method"`
	Verification string           `json:"verification"`
	Summary      CarrierSummary   `json:"summary"`
	Carrier      CarrierSpec      `json:"carrier"`
	Attempts     []CarrierAttempt `json:"attempts,omitempty"`
	Samples      []CarrierSample  `json:"samples,omitempty"`
	Warnings     []string         `json:"warnings,omitempty"`
}

func BuildCarrierReport(path string, opts CarrierOptions) (*CarrierReport, error)

type CarrierSample struct {
	TimeMS int    `json:"time_ms"`
	Time   string `json:"time"`
	Value  int    `json:"value"`
}

type CarrierSpec struct {
	PlayerID      int    `json:"player_id"`
	Attribute     int    `json:"attribute"`
	AttributeName string `json:"attribute_name,omitempty"`
	WordIndex     int    `json:"word_index"`
}

type CarrierSummary struct {
	ChecksumSamples     int    `json:"checksum_samples"`
	PlayerID            int    `json:"player_id"`
	Attribute           int    `json:"attribute"`
	AttributeName       string `json:"attribute_name,omitempty"`
	WordIndex           int    `json:"word_index"`
	AttemptCount        int    `json:"attempt_count"`
	Passed              int    `json:"passed"`
	Failed              int    `json:"failed"`
	Unknown             int    `json:"unknown"`
	Unsampled           int    `json:"unsampled"`
	FirstCarrierTimeMS  int    `json:"first_carrier_time_ms,omitempty"`
	FirstCarrierTime    string `json:"first_carrier_time,omitempty"`
	LastCarrierTimeMS   int    `json:"last_carrier_time_ms,omitempty"`
	LastCarrierTime     string `json:"last_carrier_time,omitempty"`
	MinChecksumGapMS    int    `json:"min_checksum_gap_ms,omitempty"`
	MedianChecksumGapMS int    `json:"median_checksum_gap_ms,omitempty"`
	MaxChecksumGapMS    int    `json:"max_checksum_gap_ms,omitempty"`
}

type ChapterCounts struct {
	Events    int `json:"events"`
	Actions   int `json:"actions"`
	Chat      int `json:"chat"`
	Taunts    int `json:"taunts"`
	Flares    int `json:"flares"`
	Telemetry int `json:"telemetry"`
	Resigns   int `json:"resigns"`
	Viewlocks int `json:"viewlocks"`
}

type ChatCounts struct {
	Total            int `json:"total"`
	Lobby            int `json:"lobby"`
	Backlog          int `json:"backlog"`
	Game             int `json:"game"`
	DuplicatesHidden int `json:"duplicates_hidden"`
}

type ChatExpectation struct {
	Contains string `json:"contains,omitempty"`
	Exact    string `json:"exact,omitempty"`
	MinCount int    `json:"min_count,omitempty"`
}

type ChatLine struct {
	Index          int    `json:"index"`
	Source         string `json:"source"`
	TimeMS         int    `json:"time_ms,omitempty"`
	Time           string `json:"time,omitempty"`
	PlayerID       int    `json:"player_id,omitempty"`
	PlayerName     string `json:"player_name,omitempty"`
	Channel        int    `json:"channel,omitempty"`
	ChannelName    string `json:"channel_name,omitempty"`
	PlatformIcon   string `json:"platform_icon,omitempty"`
	Text           string `json:"text"`
	TauntNumber    int    `json:"taunt_number,omitempty"`
	TauntText      string `json:"taunt_text,omitempty"`
	DestinationMap int    `json:"destination_map,omitempty"`
	MessageAGP     string `json:"message_agp,omitempty"`
	SourceOffset   int    `json:"source_offset,omitempty"`
	Confidence     string `json:"confidence"`
}

type ChatReport struct {
	Path         string                 `json:"path,omitempty"`
	Method       string                 `json:"method"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Players      []FeedbackPlayer       `json:"players,omitempty"`
	Lines        []ChatLine             `json:"lines"`
	Counts       ChatCounts             `json:"counts"`
	Warnings     []string               `json:"warnings,omitempty"`
}

func ExtractChat(path string) (*ChatReport, error)

type ChecksumPhaseAllReport struct {
	Path         string                  `json:"path,omitempty"`
	Method       string                  `json:"method"`
	Verification string                  `json:"verification"`
	Summary      ChecksumPhaseAllSummary `json:"summary"`
	Words        []*ChecksumPhaseReport  `json:"words"`
}

func BuildChecksumPhaseAll(path string, opts ChecksumPhaseOptions) (*ChecksumPhaseAllReport, error)

type ChecksumPhaseAllSummary struct {
	ChecksumSamples int `json:"checksum_samples"`
	Players         int `json:"players"`
	Words           int `json:"words"`
}

type ChecksumPhaseOptions struct {
	WordIndex int
	PlayerID  int
}

type ChecksumPhasePlayer struct {
	PlayerID                int                     `json:"player_id"`
	Samples                 int                     `json:"samples"`
	DistinctValues          int                     `json:"distinct_values"`
	KnownStateChanges       int                     `json:"known_state_changes"`
	Transitions             int                     `json:"transitions"`
	DominantTransitions     int                     `json:"dominant_transitions"`
	DominantTransitionShare float64                 `json:"dominant_transition_share"`
	RepeatTransitions       int                     `json:"repeat_transitions"`
	AmbiguousFromValues     int                     `json:"ambiguous_from_values"`
	Ring                    *ChecksumWordRing       `json:"ring,omitempty"`
	Values                  []ChecksumWordValue     `json:"values,omitempty"`
	Successors              []ChecksumWordSuccessor `json:"successors,omitempty"`
}

type ChecksumPhaseReport struct {
	Path          string                `json:"path,omitempty"`
	Method        string                `json:"method"`
	Verification  string                `json:"verification"`
	Summary       ChecksumPhaseSummary  `json:"summary"`
	Players       []ChecksumPhasePlayer `json:"players,omitempty"`
	Warnings      []string              `json:"warnings,omitempty"`
	WordSemantics map[string]string     `json:"checksum_word_semantics,omitempty"`
}

func BuildChecksumPhase(path string, opts ChecksumPhaseOptions) (*ChecksumPhaseReport, error)

type ChecksumPhaseSummary struct {
	WordIndex int    `json:"word_index"`
	WordName  string `json:"word_name"`
	Samples   int    `json:"checksum_samples"`
	Players   int    `json:"players"`
}

type ChecksumProbeCandidate struct {
	PlayerID         int    `json:"player_id"`
	PlayerLabel      string `json:"player_label"`
	FromTimeMS       int    `json:"from_time_ms"`
	FromTime         string `json:"from_time"`
	ToTimeMS         int    `json:"to_time_ms"`
	ToTime           string `json:"to_time"`
	ObjectCountDelta int64  `json:"object_count_delta,omitempty"`
	UnitTypeSumDelta int64  `json:"unit_type_sum_delta,omitempty"`
	ObjectIDSumDelta int64  `json:"object_id_sum_delta,omitempty"`
	PositionSumDelta int64  `json:"position_sum_delta,omitempty"`
	Word3Delta       int64  `json:"word_3_delta,omitempty"`
	Word4Delta       int64  `json:"word_4_delta,omitempty"`
	ScoreDelta       int64  `json:"word_9_score_candidate_delta,omitempty"`
	Kind             string `json:"kind"`
	UnitID           int    `json:"unit_id,omitempty"`
	UnitName         string `json:"unit_name,omitempty"`
	ObjectID         int64  `json:"object_id,omitempty"`
	Confidence       string `json:"confidence"`
}

type ChecksumProbeOptions struct {
	Preset           string
	PlayerID         int
	ObjectCountDelta *int64
	UnitTypeSumDelta *int64
	ObjectIDSumDelta *int64
	Word3Delta       *int64
	Word4Delta       *int64
	ScoreDelta       *int64
	Limit            int
}

type ChecksumProbeReport struct {
	Path         string                `json:"path,omitempty"`
	Method       string                `json:"method"`
	Verification string                `json:"verification"`
	Summary      ChecksumProbeSummary  `json:"summary"`
	Probes       []ChecksumProbeResult `json:"probes,omitempty"`
	Warnings     []string              `json:"warnings,omitempty"`
}

func BuildChecksumProbe(path string, opts ChecksumProbeOptions) (*ChecksumProbeReport, error)

type ChecksumProbeResult struct {
	ChecksumProbeSpec
	Matched    bool                     `json:"matched"`
	MatchCount int                      `json:"match_count"`
	Result     string                   `json:"result"`
	Candidates []ChecksumProbeCandidate `json:"candidates,omitempty"`
}

type ChecksumProbeSpec struct {
	Name             string `json:"name"`
	Description      string `json:"description,omitempty"`
	PlayerID         int    `json:"player_id,omitempty"`
	ObjectCountDelta *int64 `json:"object_count_delta,omitempty"`
	UnitTypeSumDelta *int64 `json:"unit_type_sum_delta,omitempty"`
	ObjectIDSumDelta *int64 `json:"object_id_sum_delta,omitempty"`
	Word3Delta       *int64 `json:"word_3_delta,omitempty"`
	Word4Delta       *int64 `json:"word_4_delta,omitempty"`
	ScoreDelta       *int64 `json:"word_9_score_candidate_delta,omitempty"`
}

type ChecksumProbeSummary struct {
	ChecksumSamples int    `json:"checksum_samples"`
	StateDeltas     int    `json:"state_deltas"`
	ProbeCount      int    `json:"probe_count"`
	MatchedProbes   int    `json:"matched_probes"`
	DurationMS      int    `json:"duration_ms"`
	Duration        string `json:"duration"`
	Preset          string `json:"preset,omitempty"`
}

type ChecksumWordRing struct {
	Detected           bool     `json:"detected"`
	Length             int      `json:"length"`
	Values             []uint32 `json:"values_u32,omitempty"`
	ValuesSigned       []int32  `json:"values_i32,omitempty"`
	ForwardSkips       int      `json:"forward_skips"`
	ReverseTransitions int      `json:"reverse_transitions"`
	OffRingTransitions int      `json:"off_ring_transitions"`
}

type ChecksumWordSuccessor struct {
	FromU32     uint32  `json:"from_u32"`
	FromI32     int32   `json:"from_i32"`
	ToU32       uint32  `json:"to_u32"`
	ToI32       int32   `json:"to_i32"`
	Count       int     `json:"count"`
	Dominant    bool    `json:"dominant"`
	ShareOfFrom float64 `json:"share_of_from"`
}

type ChecksumWordValue struct {
	U32   uint32 `json:"u32"`
	I32   int32  `json:"i32"`
	Count int    `json:"count"`
}

type ClusterSample struct {
	Path      string `json:"path"`
	SpanName  string `json:"span_name"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
	Bytes     int    `json:"bytes"`
	HexSample string `json:"hex_sample,omitempty"`
}

type CombatDeathEvent struct {
	PlayerID         int    `json:"player_id"`
	PlayerName       string `json:"player_name,omitempty"`
	FromTimeMS       int    `json:"from_time_ms"`
	FromTime         string `json:"from_time"`
	ToTimeMS         int    `json:"to_time_ms"`
	ToTime           string `json:"to_time"`
	LiveUnitID       int    `json:"live_unit_id,omitempty"`
	LiveUnitName     string `json:"live_unit_name,omitempty"`
	CorpseUnitID     int    `json:"corpse_unit_id,omitempty"`
	CorpseUnitName   string `json:"corpse_unit_name,omitempty"`
	ObjectCountDelta int64  `json:"object_count_delta"`
	UnitTypeSumDelta int64  `json:"unit_type_sum_delta"`
	ObjectIDSumDelta int64  `json:"object_id_sum_delta"`
	Word3Delta       int64  `json:"word_3_delta,omitempty"`
	Word4Delta       int64  `json:"word_4_delta,omitempty"`
	Confidence       string `json:"confidence"`
}

type CombatLossPulse struct {
	PlayerID             int                   `json:"player_id"`
	PlayerName           string                `json:"player_name,omitempty"`
	FromTimeMS           int                   `json:"from_time_ms"`
	FromTime             string                `json:"from_time"`
	ToTimeMS             int                   `json:"to_time_ms"`
	ToTime               string                `json:"to_time"`
	NetObjectsLost       int                   `json:"net_objects_lost"`
	ObjectCountDelta     int64                 `json:"object_count_delta"`
	UnitTypeSumDelta     int64                 `json:"unit_type_sum_delta"`
	ObjectIDSumDelta     int64                 `json:"object_id_sum_delta"`
	PositionSumDelta     int64                 `json:"position_sum_delta,omitempty"`
	Word3Delta           int64                 `json:"word_3_delta,omitempty"`
	SingleObjectRemoval  bool                  `json:"single_object_removal,omitempty"`
	RemovedUnitID        int                   `json:"removed_unit_id,omitempty"`
	RemovedUnitName      string                `json:"removed_unit_name,omitempty"`
	RemovedObjectID      int64                 `json:"removed_object_id,omitempty"`
	CandidatePressure    []CombatPressureEvent `json:"candidate_pressure,omitempty"`
	CandidatePressureNum int                   `json:"candidate_pressure_events"`
	Confidence           string                `json:"confidence"`
}

type CombatOptions struct {
	WindowMS   int
	Limit      int
	MinNetLoss int
	Sort       string
}

type CombatPlayerSummary struct {
	PlayerID             int                       `json:"player_id"`
	PlayerName           string                    `json:"player_name,omitempty"`
	LossPulses           int                       `json:"loss_pulses"`
	NetObjectsLost       int                       `json:"net_objects_lost"`
	SingleObjectRemovals int                       `json:"single_object_removals"`
	LargestNetLoss       int                       `json:"largest_net_loss"`
	FirstLossTimeMS      int                       `json:"first_loss_time_ms,omitempty"`
	FirstLossTime        string                    `json:"first_loss_time,omitempty"`
	FirstDeathTimeMS     int                       `json:"first_death_time_ms,omitempty"`
	FirstDeathTime       string                    `json:"first_death_time,omitempty"`
	PressureLinksAgainst int                       `json:"pressure_links_against"`
	DeathReplacements    int                       `json:"death_replacements"`
	PressureByAttacker   []CombatPressureAggressor `json:"pressure_by_attacker,omitempty"`
}

type CombatPressureAggressor struct {
	PlayerID   int    `json:"player_id"`
	PlayerName string `json:"player_name,omitempty"`
	Events     int    `json:"events"`
}

type CombatPressureEvent struct {
	TimeMS          int     `json:"time_ms"`
	Time            string  `json:"time"`
	PlayerID        int     `json:"player_id"`
	PlayerName      string  `json:"player_name,omitempty"`
	Type            string  `json:"type"`
	ActionID        int     `json:"action_id,omitempty"`
	ActionName      string  `json:"action_name,omitempty"`
	TargetObjectID  int     `json:"target_object_id,omitempty"`
	TargetOwnerID   int     `json:"target_owner_id,omitempty"`
	TargetUnitID    int     `json:"target_unit_id,omitempty"`
	TargetUnitName  string  `json:"target_unit_name,omitempty"`
	TargetClass     string  `json:"target_class,omitempty"`
	TargetX         float64 `json:"target_x,omitempty"`
	TargetY         float64 `json:"target_y,omitempty"`
	SelectedObjects int     `json:"selected_objects,omitempty"`
	Confidence      string  `json:"confidence"`
}

type CombatReport struct {
	Path         string                `json:"path,omitempty"`
	Method       string                `json:"method"`
	Verification string                `json:"verification"`
	Summary      CombatSummary         `json:"summary"`
	Players      []CombatPlayerSummary `json:"players,omitempty"`
	Targets      []CombatTargetSummary `json:"targets,omitempty"`
	DeathEvents  []CombatDeathEvent    `json:"death_events,omitempty"`
	LossPulses   []CombatLossPulse     `json:"loss_pulses,omitempty"`
	Warnings     []string              `json:"warnings,omitempty"`
}

func BuildCombatStory(path string, opts CombatOptions) (*CombatReport, error)

type CombatSummary struct {
	ChecksumSamples        int    `json:"checksum_samples"`
	StateDeltas            int    `json:"state_deltas"`
	LossPulses             int    `json:"loss_pulses"`
	ShownLossPulses        int    `json:"shown_loss_pulses"`
	DeathReplacements      int    `json:"death_replacements"`
	ShownDeathReplacements int    `json:"shown_death_replacements"`
	NetObjectsLost         int    `json:"net_objects_lost"`
	SingleObjectRemovals   int    `json:"single_object_removals"`
	FilteredArtifacts      int    `json:"filtered_checksum_artifacts"`
	PressureLinks          int    `json:"pressure_links"`
	PlayersWithLossPulses  int    `json:"players_with_loss_pulses"`
	Targets                int    `json:"targets"`
	WindowMS               int    `json:"pressure_window_ms"`
	Sort                   string `json:"sort"`
}

type CombatTargetSummary struct {
	TargetObjectID              int                       `json:"target_object_id"`
	TargetOwnerID               int                       `json:"target_owner_id"`
	TargetOwnerName             string                    `json:"target_owner_name,omitempty"`
	TargetUnitID                int                       `json:"target_unit_id,omitempty"`
	TargetUnitName              string                    `json:"target_unit_name,omitempty"`
	TargetClass                 string                    `json:"target_class,omitempty"`
	TargetX                     float64                   `json:"target_x,omitempty"`
	TargetY                     float64                   `json:"target_y,omitempty"`
	PressureEvents              int                       `json:"pressure_events"`
	FirstPressureTimeMS         int                       `json:"first_pressure_time_ms,omitempty"`
	FirstPressureTime           string                    `json:"first_pressure_time,omitempty"`
	LastPressureTimeMS          int                       `json:"last_pressure_time_ms,omitempty"`
	LastPressureTime            string                    `json:"last_pressure_time,omitempty"`
	NearbyLossPulses            int                       `json:"nearby_loss_pulses"`
	NetObjectsLostNearby        int                       `json:"net_objects_lost_nearby"`
	PressureByAttacker          []CombatPressureAggressor `json:"pressure_by_attacker,omitempty"`
	NearbyLossPulseVictimOnly   bool                      `json:"nearby_loss_pulse_victim_only"`
	NearbyLossNotObjectSpecific bool                      `json:"nearby_loss_not_object_specific"`
	Confidence                  string                    `json:"confidence"`
}

type CommandVocabulary struct {
	Key          string  `json:"key"`
	ActionID     int     `json:"action_id,omitempty"`
	ActionName   string  `json:"action_name,omitempty"`
	Type         string  `json:"type,omitempty"`
	Count        int     `json:"count"`
	SharePercent float64 `json:"share_percent"`
	Decoded      bool    `json:"decoded"`
}

type Context struct {
	Name              string          `json:"name,omitempty"`
	Regions           []ContextRegion `json:"regions,omitempty"`
	TelemetryPrefixes []string        `json:"telemetry_prefixes,omitempty"`
}

func LoadContext(path string) (*Context, error)

func (c *Context) MatchRegion(x, y float64) string

type ContextRegion struct {
	Name string    `json:"name"`
	Box  []float64 `json:"box,omitempty"`
}

type CorpseTable struct {
	SchemaVersion int              `json:"schema_version"`
	Generated     string           `json:"generated"`
	Method        string           `json:"method"`
	Rows          []CorpseTableRow `json:"rows"`
}

func LoadDefaultCorpseTable() (CorpseTable, error)

func ParseCorpseTable(data []byte) (CorpseTable, error)

func (t CorpseTable) Replacement(unitID, corpseID int) (CorpseTableRow, bool)

func (t CorpseTable) UniqueReplacementForDelta(delta int64) (CorpseTableRow, bool)

type CorpseTableRow struct {
	UnitID                  int      `json:"unit_id"`
	UnitName                string   `json:"unit_name"`
	CorpseID                int      `json:"corpse_id"`
	CorpseName              string   `json:"corpse_name"`
	Delta                   int64    `json:"delta"`
	Provenance              []string `json:"provenance,omitempty"`
	Confidence              string   `json:"confidence"`
	FixtureRef              string   `json:"fixture_ref,omitempty"`
	ObservedCarryMin        *int     `json:"observed_carry_min,omitempty"`
	ObservedCarryMax        *int     `json:"observed_carry_max,omitempty"`
	ObservedCarryDrop       *int     `json:"observed_carry_drop,omitempty"`
	ObservedDurationMS      *int     `json:"observed_duration_ms,omitempty"`
	ObservedDecayPerSecond  *float64 `json:"observed_decay_per_second,omitempty"`
	ObservedFreshCarryDelta *int     `json:"observed_fresh_carry_delta,omitempty"`
	CarryMeaning            string   `json:"carry_meaning,omitempty"`
}

type CorpseTableSummary struct {
	Rows           int    `json:"rows"`
	EngineVerified int    `json:"engine_verified_rows"`
	HeuristicRows  int    `json:"heuristic_rows"`
	Method         string `json:"method"`
}

type CorpusOptions struct {
	Recursive      bool
	IncludeZip     bool
	UnknownSamples int
}

type CorpusSample struct {
	Path string `json:"path"`
	Hex  string `json:"hex"`
}

type CorpusUnknownGroup struct {
	ActionID int            `json:"action_id"`
	Count    int            `json:"count"`
	Files    int            `json:"files"`
	Players  []int          `json:"players_seen,omitempty"`
	Shapes   []DeltaShape   `json:"payload_length_shapes,omitempty"`
	Samples  []CorpusSample `json:"samples,omitempty"`
	ByFile   map[string]int `json:"by_file,omitempty"`
}

type CoverageReport struct {
	Path        string          `json:"path,omitempty"`
	FileBytes   int             `json:"file_bytes"`
	Summary     CoverageSummary `json:"summary"`
	HeaderSpine *HeaderSpine    `json:"header_spine,omitempty"`
	Regions     []ReplayRegion  `json:"regions"`
	BodyOps     map[int]int     `json:"body_operations,omitempty"`
	Samples     []ReplayRegion  `json:"samples,omitempty"`
	Warnings    []string        `json:"warnings,omitempty"`
}

func BuildCoverage(path string) (*CoverageReport, error)

type CoverageSummary struct {
	CompressedHeaderBytes int     `json:"compressed_header_bytes"`
	InflatedHeaderBytes   int     `json:"inflated_header_bytes"`
	BodyBytes             int     `json:"body_bytes"`
	HeaderDecodedBytes    int     `json:"header_decoded_bytes"`
	HeaderOpaqueBytes     int     `json:"header_opaque_bytes"`
	HeaderDecodedPercent  float64 `json:"header_decoded_percent"`
	BodyDecodedBytes      int     `json:"body_decoded_bytes"`
	BodyOpaqueBytes       int     `json:"body_opaque_bytes"`
	BodyDecodedPercent    float64 `json:"body_decoded_percent"`
}

type DEPostgame struct {
	SourceOffset int                     `json:"source_offset"`
	TimeMS       int                     `json:"time_ms,omitempty"`
	Time         string                  `json:"time,omitempty"`
	TailBytes    int                     `json:"tail_bytes"`
	Version      uint32                  `json:"version,omitempty"`
	BlockCount   uint32                  `json:"block_count,omitempty"`
	Blocks       []DEPostgameBlock       `json:"blocks,omitempty"`
	WorldTimeMS  uint32                  `json:"world_time_ms,omitempty"`
	Leaderboards []DEPostgameLeaderboard `json:"leaderboards,omitempty"`
	RawTailHex   string                  `json:"raw_tail_hex,omitempty"`
	Confidence   string                  `json:"confidence"`
}

type DEPostgameBlock struct {
	ID         uint32 `json:"id"`
	Name       string `json:"name,omitempty"`
	Length     uint32 `json:"length"`
	Parsed     bool   `json:"parsed"`
	Confidence string `json:"confidence"`
}

type DEPostgameLeaderboard struct {
	ID      uint32                        `json:"id"`
	Unknown uint16                        `json:"unknown"`
	Players []DEPostgameLeaderboardPlayer `json:"players,omitempty"`
}

type DEPostgameLeaderboardPlayer struct {
	PlayerNumber int32 `json:"player_number"`
	Rank         int32 `json:"rank"`
	Rating       int32 `json:"rating"`
}

type DataSetExpectation struct {
	Status         string   `json:"status,omitempty"`
	ActiveDataSet  string   `json:"active_data_set,omitempty"`
	ActiveDataSets []string `json:"active_data_sets,omitempty"`
}

type DataSetIdentity struct {
	Status         string   `json:"status"`
	ActiveDataSet  string   `json:"active_data_set,omitempty"`
	ActiveDataSets []string `json:"active_data_sets,omitempty"`
	Checksum       uint32   `json:"checksum,omitempty"`
	WorkshopID     uint32   `json:"workshop_id,omitempty"`
	Source         string   `json:"source"`
	Confidence     string   `json:"confidence"`
	Error          string   `json:"error,omitempty"`
}

type DeadGap struct {
	StartMS    int    `json:"start_ms"`
	EndMS      int    `json:"end_ms"`
	Start      string `json:"start"`
	End        string `json:"end"`
	DurationMS int    `json:"duration_ms"`
	Duration   string `json:"duration"`
}

type DeathReport struct {
	Path         string             `json:"path,omitempty"`
	Method       string             `json:"method"`
	Verification string             `json:"verification"`
	Summary      DeathReportSummary `json:"summary"`
	Events       []DeathReportEvent `json:"events,omitempty"`
	Ambiguous    []DeathReportEvent `json:"ambiguous_replacements,omitempty"`
	Warnings     []string           `json:"warnings,omitempty"`
	CorpseTable  CorpseTableSummary `json:"corpse_table"`
}

func BuildDeathReport(path string, opts DeathReportOptions) (*DeathReport, error)

type DeathReportEvent struct {
	PlayerID         int    `json:"player_id"`
	PlayerLabel      string `json:"player_label"`
	FromTimeMS       int    `json:"from_time_ms"`
	FromTime         string `json:"from_time"`
	ToTimeMS         int    `json:"to_time_ms"`
	ToTime           string `json:"to_time"`
	LiveUnitID       int    `json:"live_unit_id,omitempty"`
	LiveUnitName     string `json:"live_unit_name,omitempty"`
	CorpseUnitID     int    `json:"corpse_unit_id,omitempty"`
	CorpseUnitName   string `json:"corpse_unit_name,omitempty"`
	UnitTypeSumDelta int64  `json:"unit_type_sum_delta"`
	ObjectIDSumDelta int64  `json:"object_id_sum_delta,omitempty"`
	PositionSumDelta int64  `json:"position_sum_delta,omitempty"`
	Word3Delta       int64  `json:"word_3_delta,omitempty"`
	Word4Delta       int64  `json:"word_4_delta,omitempty"`
	FreshCorpseCarry int    `json:"fresh_corpse_carry,omitempty"`
	EstimatedDeathMS int    `json:"estimated_death_time_ms,omitempty"`
	EstimatedDeath   string `json:"estimated_death_time,omitempty"`
	DeathTimeMethod  string `json:"death_time_method,omitempty"`
	ObjectCountDelta int64  `json:"object_count_delta"`
	Kind             string `json:"kind"`
	Confidence       string `json:"confidence"`
}

type DeathReportOptions struct {
	Limit int
}

type DeathReportSummary struct {
	ChecksumSamples         int `json:"checksum_samples"`
	ReplacementEvents       int `json:"replacement_events"`
	ShownReplacementEvents  int `json:"shown_replacement_events"`
	AmbiguousFlatTransforms int `json:"ambiguous_flat_transforms"`
	ShownAmbiguous          int `json:"shown_ambiguous"`
}

type DeltaShape struct {
	DeltaMS int `json:"delta_ms"`
	Count   int `json:"count"`
}

type DiffStateOptions struct {
	Focus string
	Limit int
}

type DiffStateReport struct {
	Before       string                 `json:"before"`
	After        string                 `json:"after"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Summary      DiffStateSummary       `json:"summary"`
	Spans        []DiffStateSpan        `json:"spans"`
	Sync         *DiffSyncComparison    `json:"sync,omitempty"`
	Warnings     []string               `json:"warnings,omitempty"`
}

func BuildDiffState(beforePath string, afterPath string, opts DiffStateOptions) (*DiffStateReport, error)

type DiffStateSpan struct {
	Space         string    `json:"space"`
	Start         int       `json:"start"`
	End           int       `json:"end"`
	Bytes         int       `json:"bytes"`
	BeforeRegion  string    `json:"before_region,omitempty"`
	BeforeStatus  string    `json:"before_status,omitempty"`
	AfterRegion   string    `json:"after_region,omitempty"`
	AfterStatus   string    `json:"after_status,omitempty"`
	BeforeHex     string    `json:"before_hex,omitempty"`
	AfterHex      string    `json:"after_hex,omitempty"`
	BeforeProfile SpanStats `json:"before_profile"`
	AfterProfile  SpanStats `json:"after_profile"`
	ShapeHints    []string  `json:"shape_hints,omitempty"`
}

type DiffStateSummary struct {
	Focus                 string `json:"focus"`
	HeaderBytesBefore     int    `json:"header_bytes_before"`
	HeaderBytesAfter      int    `json:"header_bytes_after"`
	BodyBytesBefore       int    `json:"body_bytes_before"`
	BodyBytesAfter        int    `json:"body_bytes_after"`
	HeaderChangedBytes    int    `json:"header_changed_bytes"`
	BodyChangedBytes      int    `json:"body_changed_bytes"`
	ChangedSpanCount      int    `json:"changed_span_count"`
	Shown                 int    `json:"shown"`
	SameTriggerGraph      bool   `json:"same_trigger_graph"`
	TriggerGraphCheck     string `json:"trigger_graph_check"`
	ChangedOpaqueBytes    int    `json:"changed_opaque_bytes,omitempty"`
	ChangedNonOpaqueBytes int    `json:"changed_non_opaque_bytes,omitempty"`
}

type DiffSyncComparison struct {
	Verification          string                `json:"verification"`
	BeforeChecksumSamples int                   `json:"before_checksum_samples"`
	AfterChecksumSamples  int                   `json:"after_checksum_samples"`
	BeforeDurationMS      int                   `json:"before_duration_ms"`
	BeforeDuration        string                `json:"before_duration"`
	AfterDurationMS       int                   `json:"after_duration_ms"`
	AfterDuration         string                `json:"after_duration"`
	ComparablePlayers     int                   `json:"comparable_players"`
	ChangedPlayers        int                   `json:"changed_players"`
	ChangedWords          int                   `json:"changed_words"`
	PlayerDeltas          []DiffSyncPlayerDelta `json:"player_deltas,omitempty"`
	WordDeltas            []DiffSyncWordDelta   `json:"word_deltas,omitempty"`
	Warnings              []string              `json:"warnings,omitempty"`
}

type DiffSyncPlayerDelta struct {
	PlayerID               int    `json:"player_id"`
	PlayerLabel            string `json:"player_label"`
	ChangedWords           int    `json:"changed_words"`
	ObjectCountDelta       int64  `json:"object_count_delta,omitempty"`
	UnitTypeSumDelta       int64  `json:"unit_type_sum_delta,omitempty"`
	ObjectIDSumDelta       int64  `json:"object_id_sum_delta,omitempty"`
	PositionSumDelta       int64  `json:"position_sum_delta,omitempty"`
	Word3Delta             int64  `json:"word_3_delta,omitempty"`
	Word4Delta             int64  `json:"word_4_delta,omitempty"`
	ScoreCandidateDelta    int64  `json:"word_9_score_candidate_delta,omitempty"`
	ResourceStockpileDelta int64  `json:"resource_stockpile_delta,omitempty"`
}

type DiffSyncWordDelta struct {
	PlayerID    int    `json:"player_id"`
	PlayerLabel string `json:"player_label"`
	WordIndex   int    `json:"word_index"`
	WordName    string `json:"word_name"`
	Semantic    string `json:"semantic,omitempty"`
	Before      uint32 `json:"before"`
	After       uint32 `json:"after"`
	Delta       int64  `json:"delta"`
	Confidence  string `json:"confidence"`
}

type Duplicate struct {
	SHA256 string   `json:"sha256"`
	Paths  []string `json:"paths"`
}

type EffectiveArrayEntry struct {
	Index           int    `json:"index"`
	TailOffset      int    `json:"tail_offset"`
	AbsOffset       int    `json:"abs_offset"`
	Value           uint16 `json:"value"`
	ReferenceValue  uint16 `json:"reference_value"`
	Changed         bool   `json:"changed_from_reference"`
	ReplayAvailable bool   `json:"replay_available"`
	Meaning         string `json:"meaning"`
	UnitSlot        int    `json:"unit_slot"`
	UnitID          int16  `json:"unit_id,omitempty"`
	UnitName        string `json:"unit_name,omitempty"`
	DatPresent      bool   `json:"dat_present"`
	DatEnabled      *uint8 `json:"dat_enabled,omitempty"`
	Confidence      string `json:"confidence"`
}

type EffectiveArraySummary struct {
	TailOffset int    `json:"tail_offset"`
	Count      int    `json:"count"`
	Bytes      int    `json:"bytes"`
	AbsStart   int    `json:"abs_start"`
	AbsEnd     int    `json:"abs_end"`
	Kind       string `json:"kind"`
	Confidence string `json:"confidence"`
}

type EffectiveAvailabilityDiff struct {
	Compared          bool                           `json:"compared"`
	ReferencePlayer   int                            `json:"reference_player,omitempty"`
	BaselinePlayer    int                            `json:"baseline_player,omitempty"`
	ComparedEntries   int                            `json:"compared_entries,omitempty"`
	ChangedEntries    int                            `json:"changed_entries,omitempty"`
	AvailabilityFlips int                            `json:"availability_flips,omitempty"`
	TargetAvailable   int                            `json:"target_available,omitempty"`
	BaselineAvailable int                            `json:"baseline_available,omitempty"`
	Shown             int                            `json:"shown,omitempty"`
	Entries           []EffectiveAvailabilityDiffRow `json:"entries,omitempty"`
}

type EffectiveAvailabilityDiffRow struct {
	Index            int    `json:"index"`
	TailOffset       int    `json:"tail_offset"`
	TargetValue      uint16 `json:"target_value"`
	BaselineValue    uint16 `json:"baseline_value"`
	TargetMeaning    string `json:"target_meaning"`
	BaselineMeaning  string `json:"baseline_meaning"`
	AvailabilityFlip bool   `json:"availability_flip"`
	UnitSlot         int    `json:"unit_slot"`
	UnitID           int16  `json:"unit_id,omitempty"`
	UnitName         string `json:"unit_name,omitempty"`
	DatPresent       bool   `json:"dat_present,omitempty"`
	Confidence       string `json:"confidence"`
}

type EffectiveDataModCheckOptions struct {
	DatPath        string
	BaselineReplay string
	KnownTemplate  string
	TailOffset     int
	Count          int
	Limit          int
}

type EffectiveDataModCheckReport struct {
	Path              string                         `json:"path"`
	BaselineReplay    string                         `json:"baseline_replay,omitempty"`
	DatPath           string                         `json:"dat_path,omitempty"`
	Method            string                         `json:"method"`
	Verification      string                         `json:"verification"`
	Verdict           string                         `json:"verdict"`
	DataSet           DataSetIdentity                `json:"data_set_identity"`
	TemplateCheck     EffectiveTemplateCheck         `json:"template_check"`
	TargetTemplate    EffectiveDataTemplate          `json:"target_template"`
	BaselineTemplate  *EffectiveDataTemplate         `json:"baseline_template,omitempty"`
	TailComparison    EffectiveTailComparisonSummary `json:"tail_comparison,omitempty"`
	AvailabilityArray EffectiveArraySummary          `json:"availability_array"`
	AvailabilityDiff  EffectiveAvailabilityDiff      `json:"availability_diff,omitempty"`
	Warnings          []string                       `json:"warnings,omitempty"`
}

func BuildEffectiveDataModCheck(path string, opts EffectiveDataModCheckOptions) (*EffectiveDataModCheckReport, error)

type EffectiveDataOptions struct {
	DatPath       string
	Player        int
	TailOffset    int
	Count         int
	Limit         int
	IncludeAll    bool
	KnownTemplate string
}

type EffectiveDataReport struct {
	Path              string                 `json:"path"`
	DatPath           string                 `json:"dat_path,omitempty"`
	Method            string                 `json:"method"`
	Verification      string                 `json:"verification"`
	Template          EffectiveDataTemplate  `json:"template"`
	TemplateCheck     EffectiveTemplateCheck `json:"template_check"`
	AvailabilityArray EffectiveArraySummary  `json:"availability_array"`
	Players           []EffectivePlayerTable `json:"players,omitempty"`
	Warnings          []string               `json:"warnings,omitempty"`
}

func BuildEffectiveData(path string, opts EffectiveDataOptions) (*EffectiveDataReport, error)

type EffectiveDataTemplate struct {
	ReferencePlayer       int    `json:"reference_player"`
	ReferenceLabel        string `json:"reference_label,omitempty"`
	TailStart             int    `json:"tail_start"`
	TailEnd               int    `json:"tail_end"`
	TailBytes             int    `json:"tail_bytes"`
	SHA256                string `json:"sha256"`
	KnownTemplateStatus   string `json:"known_template_status,omitempty"`
	KnownTemplateExpected string `json:"known_template_expected,omitempty"`
}

type EffectivePlayerTable struct {
	PlayerIndex    int                   `json:"player_index"`
	PlayerNumber   int                   `json:"player_number,omitempty"`
	PlayerName     string                `json:"player_name,omitempty"`
	CivID          int                   `json:"civ_id,omitempty"`
	CivName        string                `json:"civ_name,omitempty"`
	TailStart      int                   `json:"tail_start"`
	TailEnd        int                   `json:"tail_end"`
	TailBytes      int                   `json:"tail_bytes"`
	IdenticalRatio float64               `json:"identical_ratio_vs_template"`
	DiffRuns       int                   `json:"diff_runs_vs_template"`
	EntryCount     int                   `json:"entry_count"`
	ChangedEntries int                   `json:"changed_entries_vs_template"`
	Available      int                   `json:"available_entries"`
	Unavailable    int                   `json:"unavailable_entries"`
	Level0         int                   `json:"level_0_entries"`
	Level1         int                   `json:"level_1_entries"`
	Level2         int                   `json:"level_2_entries"`
	Level3         int                   `json:"level_3_entries"`
	OtherValues    int                   `json:"other_value_entries"`
	DatNamed       int                   `json:"dat_named_entries,omitempty"`
	Shown          int                   `json:"shown"`
	Entries        []EffectiveArrayEntry `json:"entries,omitempty"`
	Warnings       []string              `json:"warnings,omitempty"`
}

type EffectiveTailComparisonSummary struct {
	ComparedPlayers  int                       `json:"compared_players"`
	EqualPlayers     int                       `json:"equal_players"`
	DifferentPlayers int                       `json:"different_players"`
	MissingPlayers   int                       `json:"missing_players"`
	ComparedBytes    int                       `json:"compared_bytes"`
	ChangedBytes     int                       `json:"changed_bytes"`
	DiffRuns         int                       `json:"diff_runs"`
	Players          []EffectiveTailPlayerDiff `json:"players,omitempty"`
}

type EffectiveTailPlayerDiff struct {
	PlayerIndex    int     `json:"player_index"`
	Label          string  `json:"label,omitempty"`
	TargetBytes    int     `json:"target_bytes"`
	BaselineBytes  int     `json:"baseline_bytes"`
	SHA256         string  `json:"sha256"`
	BaselineSHA256 string  `json:"baseline_sha256"`
	Equal          bool    `json:"equal"`
	ComparedBytes  int     `json:"compared_bytes"`
	ChangedBytes   int     `json:"changed_bytes"`
	IdenticalRatio float64 `json:"identical_ratio"`
	DiffRuns       int     `json:"diff_runs"`
}

type EffectiveTemplateCheck struct {
	Status         string `json:"status"`
	BaselineName   string `json:"baseline_name"`
	ExpectedSHA256 string `json:"expected_sha256"`
	ObservedSHA256 string `json:"observed_sha256"`
	Confidence     string `json:"confidence"`
	Note           string `json:"note,omitempty"`
}

type EffectiveUnitStatDiffRow struct {
	PlayerIndex        int    `json:"player_index"`
	PlayerName         string `json:"player_name,omitempty"`
	CivID              int    `json:"civ_id"`
	CivName            string `json:"civ_name,omitempty"`
	UnitSlot           int    `json:"unit_slot"`
	UnitID             int16  `json:"unit_id"`
	UnitName           string `json:"unit_name"`
	UnitType           int    `json:"unit_type"`
	NameTailOffset     int    `json:"name_tail_offset"`
	NameOccurrences    int    `json:"name_occurrences"`
	HitPointOffset     int    `json:"hit_point_tail_offset"`
	VanillaHitPoints   int16  `json:"vanilla_hit_points"`
	EffectiveHitPoints int16  `json:"effective_hit_points"`
	HitPointDelta      int    `json:"hit_point_delta"`
	Changed            bool   `json:"changed"`
	CivSpan            int    `json:"civ_span,omitempty"`
	Classification     string `json:"classification,omitempty"`
	Confidence         string `json:"confidence"`
}

type EffectiveUnitStatsOptions struct {
	DatPath               string
	Player                int
	Limit                 int
	IncludeUnchanged      bool
	IncludeAllTypes       bool
	IncludeDuplicateNames bool
}

type EffectiveUnitStatsPlayer struct {
	PlayerIndex          int     `json:"player_index"`
	PlayerName           string  `json:"player_name,omitempty"`
	CivID                int     `json:"civ_id,omitempty"`
	CivName              string  `json:"civ_name,omitempty"`
	TailBytes            int     `json:"tail_bytes"`
	IdenticalRatioVsRef  float64 `json:"identical_ratio_vs_reference"`
	DiffRunsVsRef        int     `json:"diff_runs_vs_reference"`
	UnitsConsidered      int     `json:"units_considered"`
	Rows                 int     `json:"rows"`
	ChangedHitPoints     int     `json:"changed_hit_points"`
	DuplicateNameAnchors int     `json:"duplicate_name_anchors"`
	MissingNameAnchors   int     `json:"missing_name_anchors"`
	InvalidHitPointReads int     `json:"invalid_hit_point_reads"`
}

type EffectiveUnitStatsReport struct {
	Path           string                     `json:"path"`
	DatPath        string                     `json:"dat_path,omitempty"`
	Method         string                     `json:"method"`
	Verification   string                     `json:"verification"`
	TemplateCheck  EffectiveTemplateCheck     `json:"template_check"`
	TargetTemplate EffectiveDataTemplate      `json:"target_template"`
	DecodedFields  []string                   `json:"decoded_fields"`
	NotDecoded     []string                   `json:"not_decoded"`
	Summary        EffectiveUnitStatsSummary  `json:"summary"`
	Players        []EffectiveUnitStatsPlayer `json:"players,omitempty"`
	Rows           []EffectiveUnitStatDiffRow `json:"rows,omitempty"`
	Warnings       []string                   `json:"warnings,omitempty"`
}

func BuildEffectiveUnitStats(path string, opts EffectiveUnitStatsOptions) (*EffectiveUnitStatsReport, error)

type EffectiveUnitStatsSummary struct {
	Players              int `json:"players"`
	UnitsConsidered      int `json:"units_considered"`
	Rows                 int `json:"rows"`
	ChangedHitPoints     int `json:"changed_hit_points"`
	ModConfirmedMultiCiv int `json:"mod_confirmed_multi_civ"`
	UnresolvedSingleCiv  int `json:"unresolved_single_civ"`
	DuplicateNameAnchors int `json:"duplicate_name_anchors"`
	MissingNameAnchors   int `json:"missing_name_anchors"`
	InvalidHitPointReads int `json:"invalid_hit_point_reads"`
}

type EventCounts struct {
	Total          int            `json:"total"`
	DurationMS     int            `json:"duration_ms,omitempty"`
	Duration       string         `json:"duration,omitempty"`
	Chat           int            `json:"chat"`
	BacklogChat    int            `json:"backlog_chat,omitempty"`
	Taunts         int            `json:"taunts"`
	Flares         int            `json:"flares"`
	Telemetry      int            `json:"telemetry"`
	Syncs          int            `json:"syncs"`
	Starts         int            `json:"starts"`
	Ends           int            `json:"ends"`
	Postgames      int            `json:"postgames"`
	Saves          int            `json:"saves"`
	EmbeddedTails  int            `json:"embedded_tails"`
	Viewlocks      int            `json:"viewlocks"`
	Actions        int            `json:"actions"`
	UntypedActions int            `json:"untyped_actions"`
	SpatialActions int            `json:"spatial_actions"`
	Resigns        int            `json:"resigns"`
	UnknownOps     int            `json:"unknown_ops"`
	ActionIDs      map[int]int    `json:"action_ids,omitempty"`
	EventTypes     map[string]int `json:"event_types,omitempty"`
}

func CountReplayEvents(events []ReplayEvent) EventCounts

type EventOptions struct {
	IncludeSystemEvents  bool
	IncludeUntypedAction bool
	IncludeRaw           bool
	IncludeObjectIndex   bool
	TelemetryPrefixes    []string
	// PriorPaths supplies recordings whose early chat is evidence for DE's
	// carry-over backlog. It is intentionally opt-in.
	PriorPaths []string
}

type EventReport struct {
	Path     string           `json:"path,omitempty"`
	Method   string           `json:"method"`
	DataSet  DataSetIdentity  `json:"data_set_identity"`
	Players  []FeedbackPlayer `json:"players,omitempty"`
	Events   []ReplayEvent    `json:"events"`
	Counts   EventCounts      `json:"counts"`
	Result   MatchResult      `json:"result"`
	Warnings []string         `json:"warnings,omitempty"`
}

func ExtractEvents(path string, opts EventOptions) (*EventReport, error)

type FallbackFingerprint struct {
	Tier     string   `json:"tier"`
	SHA256   string   `json:"sha256"`
	Method   string   `json:"method"`
	Map      *MapInfo `json:"map,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

type FeedbackCluster struct {
	Key        string `json:"key"`
	Region     string `json:"region,omitempty"`
	PlayerID   int    `json:"player_id,omitempty"`
	PlayerName string `json:"player_name,omitempty"`
	Count      int    `json:"count"`
	FirstTime  string `json:"first_time,omitempty"`
	LastTime   string `json:"last_time,omitempty"`
}

type FeedbackCounts struct {
	Chat   int `json:"chat"`
	Taunts int `json:"taunts"`
	Flares int `json:"flares"`
}

type FeedbackEvent struct {
	Index          int     `json:"index"`
	Kind           string  `json:"kind"`
	TimeMS         int     `json:"time_ms,omitempty"`
	Time           string  `json:"time,omitempty"`
	Phase          string  `json:"phase,omitempty"`
	PlayerID       int     `json:"player_id,omitempty"`
	PlayerName     string  `json:"player_name,omitempty"`
	Channel        int     `json:"channel,omitempty"`
	ChannelName    string  `json:"channel_name,omitempty"`
	Text           string  `json:"text,omitempty"`
	Raw            string  `json:"raw,omitempty"`
	TauntNumber    int     `json:"taunt_number,omitempty"`
	TauntText      string  `json:"taunt_text,omitempty"`
	DestinationMap int     `json:"destination_map,omitempty"`
	MessageAGP     string  `json:"message_agp,omitempty"`
	X              float64 `json:"x,omitempty"`
	Y              float64 `json:"y,omitempty"`
	Targets        []int   `json:"targets,omitempty"`
}

func ExtractActionStreamFeedback(body []byte) ([]FeedbackEvent, map[int]string, []string)

func ExtractFeedbackEvents(body []byte) ([]FeedbackEvent, map[int]string, []string)

type FeedbackPlayer struct {
	PlayerID int    `json:"player_id"`
	Name     string `json:"name,omitempty"`
	Human    bool   `json:"human"`
	Kind     string `json:"kind,omitempty"`
}

type FeedbackReport struct {
	Path      string                     `json:"path,omitempty"`
	Method    string                     `json:"method"`
	Players   []FeedbackPlayer           `json:"players,omitempty"`
	Timeline  []FeedbackEvent            `json:"timeline"`
	PerPlayer map[string][]FeedbackEvent `json:"per_player"`
	Taunts    []TauntCount               `json:"taunts,omitempty"`
	Counts    FeedbackCounts             `json:"counts"`
	Warnings  []string                   `json:"warnings,omitempty"`
}

func ExtractFeedback(path string) (*FeedbackReport, error)

func ExtractFeedbackWithOptions(path string, opts EventOptions) (*FeedbackReport, error)

type File struct {
	Path             string               `json:"path,omitempty"`
	RecordSHA256     string               `json:"record_sha256,omitempty"`
	HeaderLength     int                  `json:"header_length"`
	InflatedBytes    int                  `json:"inflated_header_bytes"`
	GameVersion      string               `json:"game_version"`
	SaveVersion      float64              `json:"save_version"`
	LogVersion       uint32               `json:"log_version"`
	TriggerRegion    *triggergraph.Region `json:"trigger_region,omitempty"`
	TriggerGraph     *triggergraph.Graph  `json:"trigger_graph,omitempty"`
	TriggerGraphTier string               `json:"trigger_graph_decode_tier,omitempty"`
	TriggerGraphOK   bool                 `json:"trigger_graph_ok"`
	TriggerGraphErr  string               `json:"trigger_graph_error,omitempty"`
	Fallback         *FallbackFingerprint `json:"fallback,omitempty"`
	FallbackOK       bool                 `json:"fallback_ok"`
	FallbackErr      string               `json:"fallback_error,omitempty"`
	AILoadout        *aifile.Loadout      `json:"ai_loadout,omitempty"`
	DataSet          DataSetIdentity      `json:"data_set_identity"`
	ScenarioName     string               `json:"scenario_name,omitempty"`
	ScenarioDesc     string               `json:"scenario_description,omitempty"`
	ScenarioMeta     *ScenarioMetadata    `json:"scenario_metadata,omitempty"`
	HeaderMeta       *HeaderMetadata      `json:"header_metadata,omitempty"`
	LobbySettings    *LobbySettings       `json:"lobby_settings,omitempty"`
	Players          []PlayerSlot         `json:"players,omitempty"`
	AIParseErr       string               `json:"ai_parse_error,omitempty"`

	// Has unexported fields.
}

func Open(path string) (*File, error)

func OpenMetadata(path string) (*File, error)
    OpenMetadata reads bounded windows, not the complete initial-state snapshot.
    Consumers needing snapshot bytes must use ReplayCache.HeaderReader instead.

func Parse(data []byte) (*File, error)

func (f *File) HeaderBytes() []byte

type FrontierBucket struct {
	Name             string       `json:"name"`
	Bytes            int          `json:"bytes"`
	Spans            int          `json:"spans"`
	Percent          float64      `json:"percent_of_header_opaque"`
	TemplateHitBytes int          `json:"template_hit_bytes,omitempty"`
	ResidualBytes    int          `json:"residual_bytes"`
	Confidence       string       `json:"confidence"`
	Hypothesis       string       `json:"hypothesis"`
	NextMove         string       `json:"next_move"`
	Largest          []OpaqueSpan `json:"largest,omitempty"`
	TextSamples      []string     `json:"text_samples,omitempty"`
}

type FrontierDuplicate struct {
	SHA256        string                  `json:"sha256"`
	Bytes         int                     `json:"bytes"`
	Count         int                     `json:"count"`
	RepeatedBytes int                     `json:"repeated_bytes"`
	Buckets       []string                `json:"buckets,omitempty"`
	Spans         []FrontierDuplicateSpan `json:"spans"`
	HexSample     string                  `json:"hex_sample,omitempty"`
	TextSamples   []string                `json:"text_samples,omitempty"`
}

type FrontierDuplicateSpan struct {
	Space  string `json:"space"`
	Name   string `json:"name"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
	Bucket string `json:"bucket"`
}

type FrontierOptions struct {
	MinDuplicateBytes int
	Limit             int
}

type FrontierReport struct {
	Path         string                 `json:"path,omitempty"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Summary      FrontierSummary        `json:"summary"`
	Buckets      []FrontierBucket       `json:"buckets"`
	Duplicates   []FrontierDuplicate    `json:"duplicates,omitempty"`
	TemplateHits []FrontierTemplateHit  `json:"template_hits,omitempty"`
	Warnings     []string               `json:"warnings,omitempty"`
}

func BuildFrontier(path string, opts FrontierOptions) (*FrontierReport, error)

type FrontierSummary struct {
	InflatedHeaderBytes int     `json:"inflated_header_bytes"`
	HeaderDecodedBytes  int     `json:"header_decoded_bytes"`
	HeaderOpaqueBytes   int     `json:"header_opaque_bytes"`
	HeaderDecodedPct    float64 `json:"header_decoded_percent"`
	BodyBytes           int     `json:"body_bytes"`
	BodyOpaqueBytes     int     `json:"body_opaque_bytes"`
	OpaqueSpans         int     `json:"opaque_spans"`
	DuplicateGroups     int     `json:"duplicate_groups"`
	ShownDuplicates     int     `json:"shown_duplicates"`
	DuplicateBytes      int     `json:"duplicate_repeated_bytes"`
	TemplateHitCount    int     `json:"template_hit_count"`
	ShownTemplateHits   int     `json:"shown_template_hits"`
	TemplateHitBytes    int     `json:"template_hit_bytes"`
}

type FrontierTemplateHit struct {
	Space              string `json:"space"`
	Name               string `json:"name"`
	Start              int    `json:"start"`
	End                int    `json:"end"`
	Bytes              int    `json:"bytes"`
	Bucket             string `json:"bucket"`
	SourceRegion       string `json:"source_region"`
	SourceStart        int    `json:"source_start"`
	SourceEnd          int    `json:"source_end"`
	SourceRelative     int    `json:"source_relative_offset"`
	Confidence         string `json:"confidence"`
	MatchedPayloadSHA  string `json:"matched_payload_sha256"`
	MatchedPayloadHead string `json:"matched_payload_hex_sample,omitempty"`
}

type HeaderAnchorsOptions struct {
	Limit int
}

type HeaderAnchorsReport struct {
	Path         string                 `json:"path,omitempty"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Summary      HeaderAnchorsSummary   `json:"summary"`
	Captions     []HeaderCaptionAnchor  `json:"captions,omitempty"`
	Resources    []PlayerResourceBlock  `json:"player_resource_blocks,omitempty"`
	Warnings     []string               `json:"warnings,omitempty"`
}

func BuildHeaderAnchors(path string, opts HeaderAnchorsOptions) (*HeaderAnchorsReport, error)

type HeaderAnchorsSummary struct {
	InflatedHeaderBytes        int `json:"inflated_header_bytes"`
	CaptionAnchors             int `json:"caption_anchors"`
	PlayerResourceBlocks       int `json:"player_resource_blocks"`
	ShownCaptionAnchors        int `json:"shown_caption_anchors"`
	ShownPlayerResourceBlocks  int `json:"shown_player_resource_blocks"`
	CaptionMedianStrideBytes   int `json:"caption_median_stride_bytes,omitempty"`
	CaptionMinStrideBytes      int `json:"caption_min_stride_bytes,omitempty"`
	CaptionMaxStrideBytes      int `json:"caption_max_stride_bytes,omitempty"`
	DecodedCaptionRecordBytes  int `json:"decoded_caption_record_bytes,omitempty"`
	DecodedResourceRecordBytes int `json:"decoded_resource_record_bytes,omitempty"`
}

type HeaderCaptionAnchor struct {
	MarkerStart int    `json:"marker_start"`
	TextStart   int    `json:"text_start"`
	End         int    `json:"end"`
	Bytes       int    `json:"bytes"`
	Length      int    `json:"length"`
	Text        string `json:"text"`
	HasLeadFF   bool   `json:"has_lead_ff"`
	Confidence  string `json:"confidence"`
}

type HeaderMetadata struct {
	Source         string `json:"source"`
	Confidence     string `json:"confidence"`
	TailGUIDHex    string `json:"tail_guid_hex,omitempty"`
	TailGUIDOffset int    `json:"tail_guid_offset,omitempty"`
	Error          string `json:"error,omitempty"`
}

type HeaderSpine struct {
	InitialStart          int                 `json:"initial_start"`
	InitialEnd            int                 `json:"initial_end,omitempty"`
	RestoreTime           uint32              `json:"restore_time"`
	NumParticles          uint32              `json:"num_particles"`
	Identifier            uint32              `json:"identifier"`
	PlayerCount           int                 `json:"player_count"`
	Players               []InitialPlayerSpan `json:"players,omitempty"`
	PostInitialTailStart  int                 `json:"post_initial_tail_start,omitempty"`
	TriggerStart          int                 `json:"trigger_start"`
	TriggerBoundaryCheck  string              `json:"trigger_boundary_check"`
	TriggerBoundarySource string              `json:"trigger_boundary_source"`
	Notes                 []string            `json:"notes,omitempty"`
}

type HealthAnchor struct {
	LastSync       *int   `json:"last_sync_time_ms"`
	LastAction     *int   `json:"last_action_time_ms"`
	StatedDuration *int   `json:"stated_duration_ms"`
	LastChat       *int   `json:"last_chat_time_ms"`
	LastResign     *int   `json:"last_resign_time_ms"`
	ChatGap        *int   `json:"last_chat_to_sync_gap_ms"`
	ResignGap      *int   `json:"last_resign_to_sync_gap_ms"`
	EndsCleanly    bool   `json:"ends_cleanly"`
	Termination    string `json:"termination"`
	StopOffset     int64  `json:"body_stop_offset"`
	FileStopOffset int64  `json:"file_stop_offset"`
}

type HealthLoad struct {
	PeakObjects     *uint64         `json:"peak_object_count_total"`
	PeakObjectsTime *int            `json:"peak_object_count_time_ms"`
	PeakActions     float64         `json:"peak_actions_per_min"`
	PeakActionsTime *int            `json:"peak_actions_window_from_ms"`
	FirstSlope      *float64        `json:"first_30m_objects_per_min"`
	LastSlope       *float64        `json:"last_30m_objects_per_min"`
	EndObjects      []HealthObjects `json:"end_players_by_object_count"`
	EndSampleTime   *int            `json:"end_object_sample_time_ms"`
}

type HealthObjects struct {
	PlayerID    int    `json:"player_id"`
	PlayerName  string `json:"player_name,omitempty"`
	ObjectCount uint32 `json:"object_count"`
}

type HealthOptions struct{ Window time.Duration }

type HealthReport struct {
	Schema             string            `json:"schema"`
	Path               string            `json:"path"`
	WindowMS           int               `json:"window_ms"`
	EndAnchor          HealthAnchor      `json:"end_anchor"`
	Windows            []HealthWindow    `json:"windows"`
	LoadSummary        HealthLoad        `json:"load_summary"`
	Signals            []ChatLine        `json:"signals"`
	BacklogExcluded    int               `json:"backlog_chat_excluded"`
	TriggerGraphOK     bool              `json:"trigger_graph_ok"`
	TriggerGraphStatus string            `json:"trigger_graph_status"`
	TriggersNearEnd    any               `json:"triggers_near_end"`
	WordSemantics      map[string]string `json:"checksum_word_semantics"`
	Warnings           []string          `json:"warnings"`
}

func BuildHealth(path string, opts HealthOptions) (*HealthReport, error)
    BuildHealth deliberately avoids Parse: its eager header models and graph
    search are unnecessary for body triage. No sync/action event table is
    retained.

type HealthWindow struct {
	From          int             `json:"from_ms"`
	To            int             `json:"to_ms"`
	SyncCount     int             `json:"sync_count"`
	ActionCount   int             `json:"action_count"`
	ActionsPerMin float64         `json:"actions_per_min"`
	ChatCount     int             `json:"chat_count"`
	FlareCount    int             `json:"flare_count"`
	SampleTime    *int            `json:"object_sample_time_ms"`
	Objects       []HealthObjects `json:"players"`
	ObjectTotal   *uint64         `json:"object_count_total"`
}

type InboxItem struct {
	Path            string           `json:"path"`
	RecordSHA256    string           `json:"record_sha256,omitempty"`
	DuplicateOf     string           `json:"duplicate_of,omitempty"`
	Identity        StoryIdentity    `json:"identity"`
	DataSet         DataSetIdentity  `json:"data_set_identity"`
	Players         []FeedbackPlayer `json:"players,omitempty"`
	EventCounts     EventCounts      `json:"event_counts"`
	ProfileCoverage ProfileCoverage  `json:"profile_coverage,omitempty"`
	Result          MatchResult      `json:"result"`
	Moments         []StoryMoment    `json:"moments,omitempty"`
	Brief           []string         `json:"brief,omitempty"`
	IssueCards      []IssueCard      `json:"issue_cards,omitempty"`
	Warnings        []string         `json:"warnings,omitempty"`
	Error           string           `json:"error,omitempty"`
}

type InboxOptions struct {
	ContextPath string
}

type InboxReport struct {
	Root           string               `json:"root"`
	Count          int                  `json:"count"`
	Summary        InboxSummary         `json:"summary"`
	ScenarioGroups []InboxScenarioGroup `json:"scenario_groups,omitempty"`
	IssueCards     []IssueCard          `json:"issue_cards,omitempty"`
	Duplicates     []Duplicate          `json:"duplicates,omitempty"`
	Items          []InboxItem          `json:"items"`
	Warnings       []string             `json:"warnings,omitempty"`
}

func BuildInbox(root string, opts InboxOptions) (*InboxReport, error)

type InboxScenarioGroup struct {
	Key            string         `json:"key"`
	Tier           string         `json:"tier,omitempty"`
	SHA256         string         `json:"sha256,omitempty"`
	Count          int            `json:"count"`
	Paths          []string       `json:"paths,omitempty"`
	DataSets       map[string]int `json:"data_sets,omitempty"`
	Results        map[string]int `json:"results,omitempty"`
	FeedbackEvents int            `json:"feedback_events"`
	IssueCards     int            `json:"issue_cards"`
	Warnings       []string       `json:"warnings,omitempty"`
}

type InboxSummary struct {
	Parsed                int     `json:"parsed"`
	Errors                int     `json:"errors"`
	DuplicateItems        int     `json:"duplicate_items"`
	ScenarioGroups        int     `json:"scenario_groups"`
	RecognizedScenarios   int     `json:"recognized_scenarios"`
	VanillaDataSets       int     `json:"vanilla_data_sets"`
	ModdedDataSets        int     `json:"modded_data_sets"`
	UnknownDataSets       int     `json:"unknown_data_sets"`
	Completed             int     `json:"completed"`
	WinnerKnown           int     `json:"winner_known"`
	FeedbackEvents        int     `json:"feedback_events"`
	IssueCards            int     `json:"issue_cards"`
	ReplayActions         int     `json:"replay_actions"`
	UntypedActions        int     `json:"untyped_actions"`
	AverageDecodedPercent float64 `json:"average_decoded_percent,omitempty"`
}

type InitialPlayerSpan struct {
	Index                    int     `json:"index"`
	Label                    string  `json:"label"`
	Start                    int     `json:"start"`
	End                      int     `json:"end,omitempty"`
	Bytes                    int     `json:"bytes,omitempty"`
	Type                     byte    `json:"type"`
	Unknown                  byte    `json:"unknown"`
	Name                     string  `json:"name,omitempty"`
	NumHeaderData            uint32  `json:"num_header_data"`
	PayloadSpanStart         int     `json:"payload_span_start,omitempty"`
	PayloadSpanEnd           int     `json:"payload_span_end,omitempty"`
	PayloadSpanBytes         int     `json:"payload_span_bytes,omitempty"`
	AttributesStart          int     `json:"attributes_start,omitempty"`
	AttributesEnd            int     `json:"attributes_end,omitempty"`
	AttributesBytes          int     `json:"attributes_bytes,omitempty"`
	CameraX                  float64 `json:"camera_x,omitempty"`
	CameraY                  float64 `json:"camera_y,omitempty"`
	PostCameraUnknown        int32   `json:"post_camera_unknown,omitempty"`
	SpawnX                   uint16  `json:"spawn_x,omitempty"`
	SpawnY                   uint16  `json:"spawn_y,omitempty"`
	StartMetaByte            byte    `json:"start_meta_byte,omitempty"`
	CivilizationKey          string  `json:"civilization_key,omitempty"`
	ObjectMarkerStart        int     `json:"object_marker_start,omitempty"`
	ObjectMarkerEnd          int     `json:"object_marker_end,omitempty"`
	ObjectSpanStart          int     `json:"object_span_start,omitempty"`
	ObjectSpanEnd            int     `json:"object_span_end,omitempty"`
	ObjectSpanBytes          int     `json:"object_span_bytes,omitempty"`
	ObjectTailMarkerStart    int     `json:"object_tail_marker_start,omitempty"`
	ObjectTailMarkerEnd      int     `json:"object_tail_marker_end,omitempty"`
	ObjectCandidateBandStart int     `json:"object_candidate_band_start,omitempty"`
	ObjectCandidateBandEnd   int     `json:"object_candidate_band_end,omitempty"`
	ObjectCandidateBandBytes int     `json:"object_candidate_band_bytes,omitempty"`
	ObjectCandidateCount     int     `json:"object_candidate_count,omitempty"`
	ObjectBoundaryConfidence string  `json:"object_boundary_confidence,omitempty"`
	BoundaryMethod           string  `json:"boundary_method,omitempty"`
}

type IssueCard struct {
	ID          string          `json:"id"`
	Key         string          `json:"key"`
	Replay      string          `json:"replay,omitempty"`
	Scenario    StoryIdentity   `json:"scenario,omitempty"`
	Chapter     string          `json:"chapter,omitempty"`
	Region      string          `json:"region,omitempty"`
	PlayerID    int             `json:"player_id,omitempty"`
	PlayerName  string          `json:"player_name,omitempty"`
	Type        string          `json:"type"`
	Text        string          `json:"text"`
	Count       int             `json:"count"`
	FirstTime   string          `json:"first_time,omitempty"`
	LastTime    string          `json:"last_time,omitempty"`
	FirstTimeMS int             `json:"first_time_ms,omitempty"`
	LastTimeMS  int             `json:"last_time_ms,omitempty"`
	X           float64         `json:"x,omitempty"`
	Y           float64         `json:"y,omitempty"`
	Source      string          `json:"source"`
	Confidence  string          `json:"confidence"`
	Evidence    []IssueEvidence `json:"evidence,omitempty"`
}

func BuildIssueCards(replayPath string, identity StoryIdentity, events []ReplayEvent, durationMS int) []IssueCard

func BuildIssueCardsFromStoryLines(replayPath string, identity StoryIdentity, lines []StoryLine, durationMS int) []IssueCard

type IssueEvidence struct {
	Replay     string  `json:"replay,omitempty"`
	Time       string  `json:"time,omitempty"`
	TimeMS     int     `json:"time_ms,omitempty"`
	PlayerID   int     `json:"player_id,omitempty"`
	PlayerName string  `json:"player_name,omitempty"`
	Region     string  `json:"region,omitempty"`
	Type       string  `json:"type"`
	Text       string  `json:"text"`
	X          float64 `json:"x,omitempty"`
	Y          float64 `json:"y,omitempty"`
	Chapter    string  `json:"chapter,omitempty"`
}

type IssueGroup struct {
	Key        string      `json:"key"`
	Region     string      `json:"region,omitempty"`
	PlayerID   int         `json:"player_id,omitempty"`
	PlayerName string      `json:"player_name,omitempty"`
	Count      int         `json:"count"`
	Items      []IssueItem `json:"items"`
}

type IssueItem struct {
	Time string  `json:"time,omitempty"`
	Type string  `json:"type"`
	Text string  `json:"text"`
	X    float64 `json:"x,omitempty"`
	Y    float64 `json:"y,omitempty"`
}

type IssuesOptions struct {
	ContextPath string
}

type IssuesReport struct {
	Path     string       `json:"path"`
	Cards    []IssueCard  `json:"cards,omitempty"`
	Groups   []IssueGroup `json:"groups"`
	Warnings []string     `json:"warnings,omitempty"`
}

func BuildIssues(path string, opts IssuesOptions) (*IssuesReport, error)

type LifecycleFirstSeen struct {
	ObjectID      int     `json:"object_id"`
	PlayerID      int     `json:"player_id"`
	PlayerLabel   string  `json:"player_label,omitempty"`
	TimeMS        int     `json:"time_ms"`
	Time          string  `json:"time"`
	EventIndex    int     `json:"event_index"`
	Type          string  `json:"type"`
	ActionID      int     `json:"action_id,omitempty"`
	ActionName    string  `json:"action_name,omitempty"`
	X             float64 `json:"x,omitempty"`
	Y             float64 `json:"y,omitempty"`
	SelectedCount int     `json:"selected_count"`
}

type LifecycleObjectCandidate struct {
	ObjectID          int      `json:"object_id"`
	PlayerID          int      `json:"player_id"`
	PlayerLabel       string   `json:"player_label,omitempty"`
	TimeMS            int      `json:"time_ms"`
	Time              string   `json:"time"`
	EventIndex        int      `json:"event_index"`
	Type              string   `json:"type"`
	ActionID          int      `json:"action_id,omitempty"`
	ActionName        string   `json:"action_name,omitempty"`
	X                 float64  `json:"x,omitempty"`
	Y                 float64  `json:"y,omitempty"`
	SelectedCount     int      `json:"selected_count"`
	SpawnX            int      `json:"spawn_x"`
	SpawnY            int      `json:"spawn_y"`
	Distance          float64  `json:"distance"`
	UnitID            int      `json:"unit_id,omitempty"`
	UnitName          string   `json:"unit_name,omitempty"`
	CandidateUnitIDs  []int    `json:"candidate_unit_ids,omitempty"`
	CandidateUnitName []string `json:"candidate_unit_names,omitempty"`
	AmbiguousUnits    int      `json:"ambiguous_units,omitempty"`
	Kind              string   `json:"kind"`
	Confidence        string   `json:"confidence"`
	Reason            string   `json:"reason"`
}

type LifecycleOptions struct {
	SpawnRadius       float64
	MaxMoveSelection  int
	MaxBuildSelection int
}

type LifecyclePlayerSummary struct {
	PlayerID             int    `json:"player_id"`
	Label                string `json:"label"`
	FirstVillagerTimeMS  int    `json:"first_villager_time_ms,omitempty"`
	FirstVillagerTime    string `json:"first_villager_time,omitempty"`
	VillagerCandidates   int    `json:"villager_candidates"`
	SpawnMatchedObjects  int    `json:"spawn_matched_objects"`
	OvermatchedNearSpawn int    `json:"overmatched_near_spawn"`
}

type LifecycleReport struct {
	Path         string                     `json:"path,omitempty"`
	Method       string                     `json:"method"`
	Verification string                     `json:"verification"`
	Summary      LifecycleSummary           `json:"summary"`
	Players      []LifecyclePlayerSummary   `json:"players,omitempty"`
	FirstSeen    []LifecycleFirstSeen       `json:"first_seen,omitempty"`
	Candidates   []LifecycleObjectCandidate `json:"candidates,omitempty"`
	Warnings     []string                   `json:"warnings,omitempty"`
}

func BuildLifecycle(path string, opts LifecycleOptions) (*LifecycleReport, error)

type LifecycleSummary struct {
	FirstSeenObjects              int `json:"first_seen_objects"`
	SpawnRecipes                  int `json:"spawn_recipes"`
	SpawnMatchedCandidates        int `json:"spawn_matched_candidates"`
	VillagerCandidates            int `json:"villager_candidates"`
	PlayersWithVillagerCandidate  int `json:"players_with_villager_candidate"`
	OvermatchedNearVillagerEvents int `json:"overmatched_near_villager_events"`
}

type LobbySettings struct {
	Source              string   `json:"source"`
	Confidence          string   `json:"confidence"`
	Build               uint32   `json:"build,omitempty"`
	Timestamp           uint32   `json:"timestamp,omitempty"`
	GameType            uint32   `json:"game_type,omitempty"`
	MapDimension        uint32   `json:"map_dimension,omitempty"`
	RMSMapID            uint32   `json:"rms_map_id,omitempty"`
	VictoryType         uint32   `json:"victory_type,omitempty"`
	VictoryValue        uint32   `json:"victory_value,omitempty"`
	StartingResources   uint32   `json:"starting_resources,omitempty"`
	StartingAge         uint32   `json:"starting_age,omitempty"`
	MapReveal           uint32   `json:"map_reveal,omitempty"`
	GameSpeed           uint32   `json:"game_speed,omitempty"`
	TreatyLength        uint32   `json:"treaty_length,omitempty"`
	PopulationLimit     uint32   `json:"population_limit,omitempty"`
	PlayerCount         uint32   `json:"player_count,omitempty"`
	Difficulty          uint8    `json:"difficulty,omitempty"`
	RandomPositions     uint8    `json:"random_positions,omitempty"`
	AllTechnologies     uint8    `json:"all_technologies,omitempty"`
	LockTeams           uint8    `json:"lock_teams,omitempty"`
	LockSpeed           uint8    `json:"lock_speed,omitempty"`
	Multiplayer         uint8    `json:"multiplayer,omitempty"`
	Cheats              uint8    `json:"cheats,omitempty"`
	RecordGame          uint8    `json:"record_game,omitempty"`
	AnimalsEnabled      uint8    `json:"animals_enabled,omitempty"`
	PredatorsEnabled    uint8    `json:"predators_enabled,omitempty"`
	Turbo               uint8    `json:"turbo,omitempty"`
	SharedExploration   uint8    `json:"shared_exploration,omitempty"`
	TeamTogether        uint8    `json:"team_together,omitempty"`
	DiplomacyType       uint8    `json:"diplomacy_type,omitempty"`
	Ranked              uint8    `json:"ranked,omitempty"`
	RawPreDLC           []uint32 `json:"raw_pre_dlc,omitempty"`
	RawPostMapDimension uint32   `json:"raw_post_map_dimension,omitempty"`
	RawPostRMSMapID     uint32   `json:"raw_post_rms_map_id,omitempty"`
	RawPreSpeed         []uint32 `json:"raw_pre_speed,omitempty"`
	RawPreDifficulty    []byte   `json:"raw_pre_difficulty,omitempty"`
	RawTeamSettings     []byte   `json:"raw_team_settings,omitempty"`
	RawPostTeams        []uint32 `json:"raw_post_teams,omitempty"`
	RawTailFlags        []byte   `json:"raw_tail_flags,omitempty"`
}

type MapInfo struct {
	Width         int       `json:"width"`
	Height        int       `json:"height"`
	TileCount     int       `json:"tile_count"`
	TerrainSHA256 string    `json:"terrain_sha256"`
	Tiles         []MapTile `json:"tiles,omitempty"`
}

type MapInfoOptions struct {
	IncludeTiles bool
}

type MapTile struct {
	X         int `json:"x"`
	Y         int `json:"y"`
	Terrain   int `json:"terrain"`
	Elevation int `json:"elevation"`
}

type MatchResult struct {
	WinnerKnown  bool          `json:"winner_known"`
	Completed    bool          `json:"completed"`
	Method       string        `json:"method"`
	Winners      []int         `json:"winners,omitempty"`
	Losers       []int         `json:"losers,omitempty"`
	Resigned     []int         `json:"resigned,omitempty"`
	DurationMS   int           `json:"duration_ms,omitempty"`
	Duration     string        `json:"duration,omitempty"`
	ResignEvents []ResignEvent `json:"resign_events,omitempty"`
	Warnings     []string      `json:"warnings,omitempty"`
}

func InferResult(players []FeedbackPlayer, events []ReplayEvent, counts EventCounts, parseWarnings []string) MatchResult

type ObjectActionState struct {
	Decoded           bool    `json:"decoded"`
	Waiting           int     `json:"waiting,omitempty"`
	CommandFlag       int     `json:"command_flag,omitempty"`
	SelectedGroupInfo int     `json:"selected_group_info,omitempty"`
	ActionType        int     `json:"action_type,omitempty"`
	FormationID       int     `json:"formation_id,omitempty"`
	FormationRow      int     `json:"formation_row,omitempty"`
	FormationCol      int     `json:"formation_col,omitempty"`
	AttackTimer       float64 `json:"attack_timer,omitempty"`
	CaptureFlag       int     `json:"capture_flag,omitempty"`
	AttackCount       int     `json:"attack_count,omitempty"`
	V68ActionBlocks   int     `json:"v68_action_blocks,omitempty"`
}

type ObjectBuildingState struct {
	Decoded                 bool    `json:"decoded"`
	Built                   int     `json:"built,omitempty"`
	BuildPoints             float64 `json:"build_points,omitempty"`
	UniqueBuildID           int     `json:"unique_build_id,omitempty"`
	Culture                 int     `json:"culture,omitempty"`
	Burning                 int     `json:"burning,omitempty"`
	LastBurnTime            int     `json:"last_burn_time,omitempty"`
	LastGarrisonTime        int     `json:"last_garrison_time,omitempty"`
	RelicCount              int     `json:"relic_count,omitempty"`
	SpecificRelicCount      int     `json:"specific_relic_count,omitempty"`
	GatherPointExists       int     `json:"gather_point_exists,omitempty"`
	GatherPointX            float64 `json:"gather_point_x,omitempty"`
	GatherPointY            float64 `json:"gather_point_y,omitempty"`
	GatherPointObjectID     int     `json:"gather_point_object_id,omitempty"`
	GatherPointUnitTypeID   int     `json:"gather_point_unit_type_id,omitempty"`
	DesolidFlag             int     `json:"desolid_flag,omitempty"`
	PendingOrder            int     `json:"pending_order,omitempty"`
	LinkedOwner             int     `json:"linked_owner,omitempty"`
	LinkedChildren          []int   `json:"linked_children,omitempty"`
	CapturedUnitCount       int     `json:"captured_unit_count,omitempty"`
	ProductionQueueCapacity int     `json:"production_queue_capacity,omitempty"`
	EndpointX               float64 `json:"endpoint_x,omitempty"`
	EndpointY               float64 `json:"endpoint_y,omitempty"`
	Endpoint2X              float64 `json:"endpoint_2_x,omitempty"`
	Endpoint2Y              float64 `json:"endpoint_2_y,omitempty"`
	GateLocked              int     `json:"gate_locked,omitempty"`
	FirstUpdateRaw          int     `json:"first_update_raw,omitempty"`
	CloseTimerRaw           int     `json:"close_timer_raw,omitempty"`
	TerrainType             int     `json:"terrain_type,omitempty"`
	SemiAsleep              int     `json:"semi_asleep,omitempty"`
	SnowFlag                int     `json:"snow_flag,omitempty"`
	BuildingTailPlausible   bool    `json:"building_tail_plausible,omitempty"`
	V68TrailerKind          int     `json:"v68_trailer_kind,omitempty"`
	V68TrailerValue         float64 `json:"v68_trailer_value,omitempty"`
	LinkedObjectIDs         []int   `json:"linked_object_ids,omitempty"`
}

type ObjectCandidate struct {
	ObjectID                   int      `json:"object_id"`
	OwnerID                    int      `json:"owner_id"`
	OwnerLabel                 string   `json:"owner_label,omitempty"`
	RecordType                 int      `json:"record_type"`
	RecordTypeName             string   `json:"record_type_name"`
	UnitID                     int      `json:"unit_id"`
	Class                      string   `json:"class"`
	HitPoints                  float64  `json:"hitpoints,omitempty"`
	ObjectState                int      `json:"object_state,omitempty"`
	Facet                      int      `json:"facet,omitempty"`
	X                          float64  `json:"x"`
	Y                          float64  `json:"y"`
	Z                          float64  `json:"z,omitempty"`
	ResourceType               int      `json:"resource_type,omitempty"`
	Amount                     float64  `json:"amount,omitempty"`
	WorkerCount                int      `json:"worker_count,omitempty"`
	CurrentDamage              int      `json:"current_damage,omitempty"`
	UnderAttack                bool     `json:"under_attack,omitempty"`
	GroupID                    int      `json:"group_id,omitempty"`
	HasObjectProps             int      `json:"has_object_props,omitempty"`
	HasSpriteList              int      `json:"has_sprite_list,omitempty"`
	SpriteEntries              int      `json:"sprite_entries,omitempty"`
	SpriteListBytes            int      `json:"sprite_list_bytes,omitempty"`
	ParticleTypes              []int    `json:"particle_types,omitempty"`
	ParticleNames              []string `json:"particle_names,omitempty"`
	DEExtensionBytes           int      `json:"de_extension_bytes,omitempty"`
	DEExtensionStrings         []string `json:"de_extension_strings,omitempty"`
	StaticTailBytes            int      `json:"static_tail_bytes,omitempty"`
	TurnSpeed                  float64  `json:"turn_speed,omitempty"`
	MovingPrefixBytes          int      `json:"moving_prefix_bytes,omitempty"`
	Angle                      float64  `json:"angle,omitempty"`
	NumPathData                int      `json:"num_path_data,omitempty"`
	HasFuturePathData          int      `json:"has_future_path_data,omitempty"`
	HasMovementData            int      `json:"has_movement_data,omitempty"`
	NumUserWaypoints           int      `json:"num_user_waypoints,omitempty"`
	HasSubstitutePosition      int      `json:"has_substitute_position,omitempty"`
	ActionCombatPrefixBytes    int      `json:"action_combat_prefix_bytes,omitempty"`
	ActionWaiting              int      `json:"action_waiting,omitempty"`
	ActionCommandFlag          int      `json:"action_command_flag,omitempty"`
	ActionSelectedGroupInfo    int      `json:"action_selected_group_info,omitempty"`
	ActionType                 int      `json:"action_type,omitempty"`
	FormationID                int      `json:"formation_id,omitempty"`
	FormationRow               int      `json:"formation_row,omitempty"`
	FormationCol               int      `json:"formation_col,omitempty"`
	AttackTimer                float64  `json:"attack_timer,omitempty"`
	CaptureFlag                int      `json:"capture_flag,omitempty"`
	AttackCount                int      `json:"attack_count,omitempty"`
	CombatPrefixBytes          int      `json:"combat_prefix_bytes,omitempty"`
	HasAI                      int      `json:"has_ai,omitempty"`
	CombatTailPrefixBytes      int      `json:"combat_tail_prefix_bytes,omitempty"`
	HasDEPosition              bool     `json:"has_de_position,omitempty"`
	TownBellFlag               int      `json:"town_bell_flag,omitempty"`
	TownBellTargetID           int      `json:"town_bell_target_id,omitempty"`
	TownBellTargetX            float64  `json:"town_bell_target_x,omitempty"`
	TownBellTargetY            float64  `json:"town_bell_target_y,omitempty"`
	TownBellAction             int      `json:"town_bell_action,omitempty"`
	BerserkerTimer             float64  `json:"berserker_timer,omitempty"`
	NumBuilders                int      `json:"num_builders,omitempty"`
	NumHealers                 int      `json:"num_healers,omitempty"`
	CombatV68TrailerBytes      int      `json:"combat_v68_trailer_bytes,omitempty"`
	CombatV68PrimaryObjectID   int      `json:"combat_v68_primary_object_id,omitempty"`
	CombatV68SecondaryObjectID int      `json:"combat_v68_secondary_object_id,omitempty"`
	CombatV68TrailerKind       int      `json:"combat_v68_trailer_kind,omitempty"`
	CombatV68TrailerCounter    int      `json:"combat_v68_trailer_counter,omitempty"`
	V68ActionBlockBytes        int      `json:"v68_action_block_bytes,omitempty"`
	V68ActionBlockCount        int      `json:"v68_action_block_count,omitempty"`
	V68PostActionStateBytes    int      `json:"v68_post_action_state_bytes,omitempty"`
	V68PositionOrderTailBytes  int      `json:"v68_position_order_tail_bytes,omitempty"`
	FinalSpanTailMarkerBytes   int      `json:"final_span_tail_marker_bytes,omitempty"`
	BuildingPrefixBytes        int      `json:"building_prefix_bytes,omitempty"`
	Built                      int      `json:"built,omitempty"`
	BuildPoints                float64  `json:"build_points,omitempty"`
	UniqueBuildID              int      `json:"unique_build_id,omitempty"`
	Culture                    int      `json:"culture,omitempty"`
	Burning                    int      `json:"burning,omitempty"`
	LastBurnTime               int      `json:"last_burn_time,omitempty"`
	LastGarrisonTime           int      `json:"last_garrison_time,omitempty"`
	RelicCount                 int      `json:"relic_count,omitempty"`
	SpecificRelicCount         int      `json:"specific_relic_count,omitempty"`
	GatherPointExists          int      `json:"gather_point_exists,omitempty"`
	GatherPointX               float64  `json:"gather_point_x,omitempty"`
	GatherPointY               float64  `json:"gather_point_y,omitempty"`
	GatherPointObjectID        int      `json:"gather_point_object_id,omitempty"`
	GatherPointUnitTypeID      int      `json:"gather_point_unit_type_id,omitempty"`
	DesolidFlag                int      `json:"desolid_flag,omitempty"`
	PendingOrder               int      `json:"pending_order,omitempty"`
	LinkedOwner                int      `json:"linked_owner,omitempty"`
	LinkedChildren             []int    `json:"linked_children,omitempty"`
	CapturedUnitCount          int      `json:"captured_unit_count,omitempty"`
	BuildingExtraActionsBytes  int      `json:"building_extra_actions_bytes,omitempty"`
	BuildingQueueHeaderBytes   int      `json:"building_queue_header_bytes,omitempty"`
	ProductionQueueCapacity    int      `json:"production_queue_capacity,omitempty"`
	BuildingTailPrefixBytes    int      `json:"building_tail_prefix_bytes,omitempty"`
	EndpointX                  float64  `json:"endpoint_x,omitempty"`
	EndpointY                  float64  `json:"endpoint_y,omitempty"`
	Endpoint2X                 float64  `json:"endpoint_2_x,omitempty"`
	Endpoint2Y                 float64  `json:"endpoint_2_y,omitempty"`
	GateLocked                 int      `json:"gate_locked,omitempty"`
	FirstUpdateRaw             int      `json:"first_update_raw,omitempty"`
	CloseTimerRaw              int      `json:"close_timer_raw,omitempty"`
	TerrainType                int      `json:"terrain_type,omitempty"`
	SemiAsleep                 int      `json:"semi_asleep,omitempty"`
	SnowFlag                   int      `json:"snow_flag,omitempty"`
	BuildingV68TrailerBytes    int      `json:"building_v68_trailer_bytes,omitempty"`
	BuildingV68TrailerFlag     int      `json:"building_v68_trailer_flag,omitempty"`
	BuildingV68TrailerValue    float64  `json:"building_v68_trailer_value,omitempty"`
	BuildingV68TrailerKind     int      `json:"building_v68_trailer_kind,omitempty"`
	BuildingV68LinkedObjectIDs []int    `json:"building_v68_linked_object_ids,omitempty"`
	ZeroTailPaddingBytes       int      `json:"zero_tail_padding_bytes,omitempty"`
	Offset                     int      `json:"offset"`
	SpanStart                  int      `json:"span_start"`
	SpanEnd                    int      `json:"span_end"`
	PrefixBytes                int      `json:"prefix_bytes"`
	NextOffset                 int      `json:"next_offset,omitempty"`
	RecordBytes                int      `json:"record_bytes,omitempty"`
	BodyBytes                  int      `json:"body_bytes,omitempty"`
	BodySampleHex              string   `json:"body_sample_hex,omitempty"`
	DecodedBodyPrefixBytes     int      `json:"decoded_body_prefix_bytes,omitempty"`
	OpaqueSampleHex            string   `json:"opaque_sample_hex,omitempty"`
	EndConfidence              string   `json:"end_confidence,omitempty"`
	CommandRefs                int      `json:"command_refs,omitempty"`
	TargetRefs                 int      `json:"target_refs,omitempty"`
	SelectedRefs               int      `json:"selected_refs,omitempty"`
	Confidence                 string   `json:"confidence"`
	Score                      int      `json:"score"`
	Ambiguous                  bool     `json:"ambiguous,omitempty"`
}

func (object ObjectCandidate) Reference() ObjectReference

type ObjectCombatState struct {
	Decoded                  bool    `json:"decoded"`
	HasAI                    int     `json:"has_ai,omitempty"`
	HasDEPosition            bool    `json:"has_de_position,omitempty"`
	TownBellFlag             int     `json:"town_bell_flag,omitempty"`
	TownBellTargetID         int     `json:"town_bell_target_id,omitempty"`
	TownBellTargetX          float64 `json:"town_bell_target_x,omitempty"`
	TownBellTargetY          float64 `json:"town_bell_target_y,omitempty"`
	TownBellAction           int     `json:"town_bell_action,omitempty"`
	BerserkerTimer           float64 `json:"berserker_timer,omitempty"`
	NumBuilders              int     `json:"num_builders,omitempty"`
	NumHealers               int     `json:"num_healers,omitempty"`
	PrimaryObjectID          int     `json:"primary_object_id,omitempty"`
	SecondaryObjectID        int     `json:"secondary_object_id,omitempty"`
	V68TrailerKind           int     `json:"v68_trailer_kind,omitempty"`
	V68TrailerCounter        int     `json:"v68_trailer_counter,omitempty"`
	PositionOrderTailDecoded bool    `json:"position_order_tail_decoded,omitempty"`
}

type ObjectIndexOptions struct {
	Limit          int
	ReferencedOnly bool
}

type ObjectIndexReport struct {
	Path         string                `json:"path,omitempty"`
	Method       string                `json:"method"`
	Verification string                `json:"verification"`
	Summary      ObjectIndexSummary    `json:"summary"`
	Players      []ObjectPlayerSummary `json:"players,omitempty"`
	Shown        int                   `json:"shown"`
	Objects      []ObjectCandidate     `json:"objects,omitempty"`
	Warnings     []string              `json:"warnings,omitempty"`
}

func BuildObjectIndex(path string, opts ObjectIndexOptions) (*ObjectIndexReport, error)

func BuildObjectIndexFromEvents(path string, events []ReplayEvent, opts ObjectIndexOptions) (*ObjectIndexReport, error)

type ObjectIndexSummary struct {
	Candidates              int `json:"candidates"`
	ReferencedCandidates    int `json:"referenced_candidates"`
	CommandReferencedIDs    int `json:"command_referenced_ids"`
	TargetReferencedIDs     int `json:"target_referenced_ids"`
	AmbiguousCandidateIDs   int `json:"ambiguous_candidate_ids"`
	UnresolvedReferencedIDs int `json:"unresolved_referenced_ids"`
	DecodedPrefixBytes      int `json:"decoded_prefix_bytes"`
	DecodedBodyPrefixBytes  int `json:"decoded_body_prefix_bytes,omitempty"`
	BoundedBodyBytes        int `json:"bounded_body_bytes"`
	OpaqueBodyBytes         int `json:"opaque_body_bytes,omitempty"`
}

type ObjectPlayerSummary struct {
	PlayerID     int            `json:"player_id"`
	Label        string         `json:"label"`
	Candidates   int            `json:"candidates"`
	Referenced   int            `json:"referenced"`
	ByClass      map[string]int `json:"by_class,omitempty"`
	ByUnitID     map[int]int    `json:"by_unit_id,omitempty"`
	ByRecordType map[int]int    `json:"by_record_type,omitempty"`
}

type ObjectReference struct {
	ObjectID       int     `json:"object_id"`
	OwnerID        int     `json:"owner_id"`
	OwnerLabel     string  `json:"owner_label,omitempty"`
	RecordType     int     `json:"record_type"`
	RecordTypeName string  `json:"record_type_name"`
	UnitID         int     `json:"unit_id"`
	Class          string  `json:"class"`
	X              float64 `json:"x"`
	Y              float64 `json:"y"`
	Confidence     string  `json:"confidence"`
}

type ObjectShape struct {
	RecordType             int         `json:"record_type"`
	RecordTypeName         string      `json:"record_type_name"`
	UnitID                 int         `json:"unit_id"`
	UnitName               string      `json:"unit_name,omitempty"`
	Class                  string      `json:"class"`
	RecordBytes            int         `json:"record_bytes"`
	PrefixBytes            int         `json:"prefix_bytes"`
	BodyBytes              int         `json:"body_bytes"`
	DecodedBodyPrefixEach  int         `json:"decoded_body_prefix_each,omitempty"`
	DecodedBodyPrefixBytes int         `json:"decoded_body_prefix_bytes,omitempty"`
	OpaqueBodyEach         int         `json:"opaque_body_each,omitempty"`
	OpaqueBodyBytes        int         `json:"opaque_body_bytes,omitempty"`
	EndConfidence          string      `json:"end_confidence"`
	Count                  int         `json:"count"`
	Referenced             int         `json:"referenced"`
	TargetRefs             int         `json:"target_refs"`
	SelectedRefs           int         `json:"selected_refs"`
	Owners                 map[int]int `json:"owners,omitempty"`
	ExampleObjectIDs       []int       `json:"example_object_ids,omitempty"`
	ExampleOffsets         []int       `json:"example_offsets,omitempty"`
	BodySampleHex          string      `json:"body_sample_hex,omitempty"`
	OpaqueSampleHex        string      `json:"opaque_sample_hex,omitempty"`
}

type ObjectShapeReport struct {
	Path         string             `json:"path,omitempty"`
	Method       string             `json:"method"`
	Verification string             `json:"verification"`
	Summary      ObjectShapeSummary `json:"summary"`
	Shown        int                `json:"shown"`
	Shapes       []ObjectShape      `json:"shapes,omitempty"`
	Warnings     []string           `json:"warnings,omitempty"`
}

func BuildObjectShapes(path string, opts ObjectIndexOptions) (*ObjectShapeReport, error)

type ObjectShapeSummary struct {
	Candidates             int `json:"candidates"`
	ReferencedCandidates   int `json:"referenced_candidates"`
	ShapeCount             int `json:"shape_count"`
	DecodedPrefixBytes     int `json:"decoded_prefix_bytes"`
	DecodedBodyPrefixBytes int `json:"decoded_body_prefix_bytes,omitempty"`
	BoundedBodyBytes       int `json:"bounded_body_bytes"`
	OpaqueBodyBytes        int `json:"opaque_body_bytes,omitempty"`
	MinRecordBytes         int `json:"min_record_bytes,omitempty"`
	MaxRecordBytes         int `json:"max_record_bytes,omitempty"`
	FinalBodyShapeCount    int `json:"final_body_shape_count,omitempty"`
	ReferencedShapeCount   int `json:"referenced_shape_count,omitempty"`
	UnreferencedShapeCount int `json:"unreferenced_shape_count,omitempty"`
}

type ObjectStateCard struct {
	ObjectID       int                  `json:"object_id"`
	OwnerID        int                  `json:"owner_id"`
	OwnerLabel     string               `json:"owner_label,omitempty"`
	UnitID         int                  `json:"unit_id"`
	UnitName       string               `json:"unit_name,omitempty"`
	Class          string               `json:"class"`
	RecordType     int                  `json:"record_type"`
	RecordTypeName string               `json:"record_type_name"`
	HitPoints      float64              `json:"hitpoints,omitempty"`
	ObjectState    int                  `json:"object_state,omitempty"`
	X              float64              `json:"x"`
	Y              float64              `json:"y"`
	CurrentDamage  int                  `json:"current_damage,omitempty"`
	UnderAttack    bool                 `json:"under_attack,omitempty"`
	CommandRefs    int                  `json:"command_refs,omitempty"`
	TargetRefs     int                  `json:"target_refs,omitempty"`
	SelectedRefs   int                  `json:"selected_refs,omitempty"`
	Action         *ObjectActionState   `json:"action,omitempty"`
	Combat         *ObjectCombatState   `json:"combat,omitempty"`
	Building       *ObjectBuildingState `json:"building,omitempty"`
	Coverage       ObjectStateCoverage  `json:"coverage"`
	Confidence     string               `json:"confidence"`
	Notes          []string             `json:"notes,omitempty"`
}

type ObjectStateCoverage struct {
	RecordBytes            int    `json:"record_bytes"`
	PrefixBytes            int    `json:"prefix_bytes"`
	BodyBytes              int    `json:"body_bytes"`
	DecodedBodyPrefixBytes int    `json:"decoded_body_prefix_bytes"`
	OpaqueBodyBytes        int    `json:"opaque_body_bytes"`
	EndConfidence          string `json:"end_confidence,omitempty"`
}

type ObjectStateOptions struct {
	Limit          int
	ReferencedOnly bool
	Class          string
}

type ObjectStateReport struct {
	Path         string                `json:"path,omitempty"`
	Method       string                `json:"method"`
	Verification string                `json:"verification"`
	Summary      ObjectStateSummary    `json:"summary"`
	Players      []ObjectPlayerSummary `json:"players,omitempty"`
	Shown        int                   `json:"shown"`
	Objects      []ObjectStateCard     `json:"objects,omitempty"`
	Warnings     []string              `json:"warnings,omitempty"`
}

func BuildObjectState(path string, opts ObjectStateOptions) (*ObjectStateReport, error)

type ObjectStateSummary struct {
	Objects                int `json:"objects"`
	Referenced             int `json:"referenced"`
	Buildings              int `json:"buildings"`
	Gates                  int `json:"gates"`
	Damaged                int `json:"damaged"`
	UnderAttack            int `json:"under_attack"`
	ActionPrefixDecoded    int `json:"action_prefix_decoded"`
	CombatTailDecoded      int `json:"combat_tail_decoded"`
	BuildingPrefixDecoded  int `json:"building_prefix_decoded"`
	BuildingTailDecoded    int `json:"building_tail_decoded"`
	BuildingTailPlausible  int `json:"building_tail_plausible"`
	ProductionQueueHeaders int `json:"production_queue_headers"`
	GatherPoints           int `json:"gather_points"`
	GateLockedNonZero      int `json:"gate_locked_nonzero"`
	V68ActionBlockObjects  int `json:"v68_action_block_objects"`
	V68TrailerObjects      int `json:"v68_trailer_objects"`
	DecodedBodyPrefixBytes int `json:"decoded_body_prefix_bytes"`
	OpaqueBodyBytes        int `json:"opaque_body_bytes"`
}

type OpaqueCluster struct {
	Key                 string          `json:"key"`
	Space               string          `json:"space"`
	NamePattern         string          `json:"name_pattern"`
	PrevPattern         string          `json:"prev_pattern,omitempty"`
	NextPattern         string          `json:"next_pattern,omitempty"`
	ByteBucket          string          `json:"byte_bucket"`
	Count               int             `json:"count"`
	Files               int             `json:"files"`
	TotalBytes          int             `json:"total_bytes"`
	MinBytes            int             `json:"min_bytes"`
	MaxBytes            int             `json:"max_bytes"`
	AvgBytes            float64         `json:"avg_bytes"`
	AvgEntropy          float64         `json:"avg_entropy"`
	AvgZeroPercent      float64         `json:"avg_zero_percent"`
	AvgFFPercent        float64         `json:"avg_ff_percent"`
	AvgPrintablePercent float64         `json:"avg_printable_percent"`
	ShapeHints          []string        `json:"shape_hints,omitempty"`
	Samples             []ClusterSample `json:"samples,omitempty"`
}

type OpaqueClusterOptions struct {
	Recursive   bool
	IncludeZip  bool
	Space       string
	MinBytes    int
	Limit       int
	SampleLimit int
}

type OpaqueClusterReport struct {
	Folder       string               `json:"folder"`
	Method       string               `json:"method"`
	Verification string               `json:"verification"`
	Summary      OpaqueClusterSummary `json:"summary"`
	Clusters     []OpaqueCluster      `json:"clusters"`
	Warnings     []string             `json:"warnings,omitempty"`
}

func BuildOpaqueClusters(folder string, opts OpaqueClusterOptions) (*OpaqueClusterReport, error)

type OpaqueClusterSummary struct {
	Files             int `json:"files"`
	Failures          int `json:"failures"`
	Spans             int `json:"spans"`
	Clusters          int `json:"clusters"`
	Shown             int `json:"shown"`
	TotalOpaqueBytes  int `json:"total_opaque_bytes"`
	HeaderOpaqueBytes int `json:"header_opaque_bytes"`
	BodyOpaqueBytes   int `json:"body_opaque_bytes"`
}

type OpaqueSpan struct {
	Space          string   `json:"space"`
	Name           string   `json:"name"`
	Start          int      `json:"start"`
	End            int      `json:"end"`
	Bytes          int      `json:"bytes"`
	Status         string   `json:"status"`
	Confidence     string   `json:"confidence"`
	Entropy        float64  `json:"entropy"`
	DistinctBytes  int      `json:"distinct_bytes"`
	ZeroBytes      int      `json:"zero_bytes"`
	FFBytes        int      `json:"ff_bytes"`
	PrintableBytes int      `json:"printable_bytes"`
	LongestZeroRun int      `json:"longest_zero_run"`
	LongestFFRun   int      `json:"longest_ff_run"`
	HexSample      string   `json:"hex_sample,omitempty"`
	TextSamples    []string `json:"text_samples,omitempty"`
	PrevRegion     string   `json:"prev_region,omitempty"`
	NextRegion     string   `json:"next_region,omitempty"`
	ShapeHints     []string `json:"shape_hints,omitempty"`
}

type OpaqueSpanOptions struct {
	Space    string
	MinBytes int
	Limit    int
}

type OpaqueSpanReport struct {
	Path         string                 `json:"path,omitempty"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Summary      OpaqueSpanSummary      `json:"summary"`
	Spans        []OpaqueSpan           `json:"spans"`
	Warnings     []string               `json:"warnings,omitempty"`
}

func BuildOpaqueSpans(path string, opts OpaqueSpanOptions) (*OpaqueSpanReport, error)

type OpaqueSpanSummary struct {
	InflatedHeaderBytes int `json:"inflated_header_bytes"`
	BodyBytes           int `json:"body_bytes"`
	HeaderOpaqueBytes   int `json:"header_opaque_bytes"`
	BodyOpaqueBytes     int `json:"body_opaque_bytes"`
	SpanCount           int `json:"span_count"`
	Shown               int `json:"shown"`
}

type OpaqueTargetReport struct {
	Path              string            `json:"path"`
	Method            string            `json:"method"`
	Verification      string            `json:"verification"`
	HeaderOpaqueBytes int               `json:"header_opaque_bytes"`
	PlayerTails       []PlayerTailIdent `json:"player_tails,omitempty"`
	CommonPrefixBytes int               `json:"cross_player_common_prefix_bytes"`
	ReducibleEstimate int               `json:"reducible_bytes_if_template_proven"`
	SelectedTarget    string            `json:"selected_target"`
	Rationale         []string          `json:"rationale,omitempty"`
	NextIncrement     string            `json:"next_parser_increment"`
	Warnings          []string          `json:"warnings,omitempty"`
}

func BuildOpaqueTarget(path string) (*OpaqueTargetReport, error)

type PlayerActionExpectation struct {
	PlayerID          int     `json:"player_id"`
	MinActions        int     `json:"min_actions,omitempty"`
	MinDecodedPercent float64 `json:"min_decoded_percent,omitempty"`
}

type PlayerEventsFilters struct {
	PlayerID int    `json:"player_id,omitempty"`
	Type     string `json:"type,omitempty"`
	ActionID *int   `json:"action_id,omitempty"`
	Phase    string `json:"phase,omitempty"`
	FromMS   int    `json:"from_ms,omitempty"`
	ToMS     int    `json:"to_ms,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

type PlayerEventsOptions struct {
	ContextPath    string
	PlayerID       int
	Type           string
	ActionID       int
	ActionIDSet    bool
	Phase          string
	FromMS         int
	ToMS           int
	Limit          int
	IncludeRaw     bool
	IncludeObjects bool
}

type PlayerEventsReport struct {
	Path         string              `json:"path,omitempty"`
	Method       string              `json:"method"`
	ContextName  string              `json:"context_name,omitempty"`
	Players      []FeedbackPlayer    `json:"players,omitempty"`
	Counts       EventCounts         `json:"counts"`
	Filters      PlayerEventsFilters `json:"filters"`
	Shown        int                 `json:"shown"`
	Events       []ReplayEvent       `json:"events"`
	Verification string              `json:"verification"`
	Claims       []StoryClaim        `json:"claims,omitempty"`
	Warnings     []string            `json:"warnings,omitempty"`
}

func BuildPlayerEvents(path string, opts PlayerEventsOptions) (*PlayerEventsReport, error)

type PlayerProfile struct {
	PlayerID       int                  `json:"player_id"`
	PlayerName     string               `json:"player_name,omitempty"`
	Kind           string               `json:"kind,omitempty"`
	Actions        int                  `json:"actions"`
	DecodedActions int                  `json:"decoded_actions"`
	Undecoded      int                  `json:"undecoded_actions"`
	APM            float64              `json:"apm"`
	APMWindows     []APMWindow          `json:"apm_windows,omitempty"`
	DeadGaps       []DeadGap            `json:"dead_gaps,omitempty"`
	Vocabulary     []CommandVocabulary  `json:"vocabulary,omitempty"`
	Spatial        SpatialProfile       `json:"spatial"`
	Feedback       ProfileFeedbackCount `json:"feedback"`
	Notables       []ProfileNotable     `json:"notables,omitempty"`
	Claims         []StoryClaim         `json:"claims,omitempty"`
	Events         []ProfileEventRow    `json:"events,omitempty"`
}

type PlayerProfileOptions struct {
	ContextPath   string
	WindowSec     int
	DeadGapSec    int
	CellSize      float64
	IncludeEvents bool
}

type PlayerProfileReport struct {
	Path         string                  `json:"path,omitempty"`
	Method       string                  `json:"method"`
	ContextName  string                  `json:"context_name,omitempty"`
	Identity     StoryIdentity           `json:"identity"`
	DataSet      DataSetIdentity         `json:"data_set_identity"`
	Players      []PlayerProfile         `json:"players"`
	Camera       []CameraObservation     `json:"camera_observed,omitempty"`
	EventCounts  EventCounts             `json:"event_counts"`
	Coverage     ProfileCoverage         `json:"coverage"`
	Result       MatchResult             `json:"result"`
	Verification string                  `json:"verification"`
	Claims       []StoryClaim            `json:"claims,omitempty"`
	Missing      []string                `json:"missing,omitempty"`
	Warnings     []string                `json:"warnings,omitempty"`
	Options      PlayerProfileRunOptions `json:"options"`
}

func BuildPlayerProfile(path string, opts PlayerProfileOptions) (*PlayerProfileReport, error)

type PlayerProfileRunOptions struct {
	WindowSec     int     `json:"window_sec"`
	DeadGapSec    int     `json:"dead_gap_sec"`
	CellSize      float64 `json:"cell_size"`
	IncludeEvents bool    `json:"include_events"`
}

type PlayerResourceBlock struct {
	Start      int                    `json:"start"`
	End        int                    `json:"end"`
	Bytes      int                    `json:"bytes"`
	Stride     int                    `json:"stride"`
	Players    []PlayerResourceRecord `json:"players"`
	Confidence string                 `json:"confidence"`
	Note       string                 `json:"note,omitempty"`
}

type PlayerResourceRecord struct {
	PlayerIndex int    `json:"player_index"`
	Player      int    `json:"player"`
	Start       int    `json:"start"`
	End         int    `json:"end"`
	Unknown0    uint32 `json:"unknown0"`
	Unknown1    uint32 `json:"unknown1"`
	Gold        uint32 `json:"gold"`
	Wood        uint32 `json:"wood"`
	Food        uint32 `json:"food"`
	Stone       uint32 `json:"stone"`
}

type PlayerSeriesOptions struct {
	PlayerID int
	// Limit caps emitted samples per player (0 = all). Aggregates always cover
	// every checksum sample and every delta.
	Limit int
	// ChangesOnly emits only samples where one of the decoded checksum words
	// changed from that player's previous sample. Aggregates are unaffected.
	ChangesOnly bool
}

type PlayerSeriesPlayer struct {
	PlayerID                int                 `json:"player_id"`
	PlayerLabel             string              `json:"player_label"`
	PlayerName              string              `json:"player_name,omitempty"`
	ProfileID               int                 `json:"profile_id,omitempty"`
	Samples                 int                 `json:"samples"`
	EmittedSamples          int                 `json:"emitted_samples"`
	FirstTimeMS             int                 `json:"first_time_ms,omitempty"`
	FirstTime               string              `json:"first_time,omitempty"`
	LastTimeMS              int                 `json:"last_time_ms,omitempty"`
	LastTime                string              `json:"last_time,omitempty"`
	FinalObjectCount        int                 `json:"final_object_count"`
	FinalUnitTypeSum        uint32              `json:"final_unit_type_sum"`
	FinalObjectIDSum        uint32              `json:"final_object_id_sum"`
	FinalResourceStockpile  uint32              `json:"final_resource_stockpile"`
	FinalPositionSum        uint32              `json:"final_position_sum"`
	FinalScoreCandidate     uint32              `json:"final_score_candidate"`
	ObjectAdds              int                 `json:"object_adds"`
	ObjectRemoves           int                 `json:"object_removes"`
	FilteredArtifactAdds    int                 `json:"filtered_artifact_adds,omitempty"`
	FilteredArtifactRemoves int                 `json:"filtered_artifact_removes,omitempty"`
	DeathReplacements       int                 `json:"death_replacements,omitempty"`
	SingleObjectAdds        int                 `json:"single_object_adds"`
	SingleObjectRemoves     int                 `json:"single_object_removes"`
	MultiObjectAddEvents    int                 `json:"multi_object_add_events"`
	MultiObjectRemoveEvents int                 `json:"multi_object_remove_events"`
	AddedUnitTypes          []UnitDeltaCount    `json:"added_unit_types,omitempty"`
	RemovedUnitTypes        []UnitDeltaCount    `json:"removed_unit_types,omitempty"`
	DeathReplacementUnits   []UnitDeltaCount    `json:"death_replacement_units,omitempty"`
	ProducedEstimate        int                 `json:"produced_estimate"`
	LostEstimate            int                 `json:"lost_estimate"`
	ProducedEstimateSource  string              `json:"produced_estimate_source"`
	LostEstimateSource      string              `json:"lost_estimate_source"`
	AttributionConfidence   string              `json:"attribution_confidence"`
	SamplesOut              []PlayerStateSample `json:"samples_out,omitempty"`
}

type PlayerSeriesReport struct {
	Path          string               `json:"path,omitempty"`
	Method        string               `json:"method"`
	Verification  string               `json:"verification"`
	Summary       PlayerSeriesSummary  `json:"summary"`
	WordSemantics map[string]string    `json:"checksum_word_semantics,omitempty"`
	Players       []PlayerSeriesPlayer `json:"players,omitempty"`
	Deltas        []SyncStateDelta     `json:"deltas,omitempty"`
	Warnings      []string             `json:"warnings,omitempty"`
}

func BuildPlayerSeries(path string, opts PlayerSeriesOptions) (*PlayerSeriesReport, error)

type PlayerSeriesSummary struct {
	ChecksumSamples int    `json:"checksum_samples"`
	Players         int    `json:"players"`
	EmittedSamples  int    `json:"emitted_samples"`
	EmittedDeltas   int    `json:"emitted_deltas"`
	DurationMS      int    `json:"duration_ms"`
	Duration        string `json:"duration"`
}

type PlayerSlot struct {
	Slot           int    `json:"slot"`
	Active         bool   `json:"active"`
	Number         int    `json:"number,omitempty"`
	Name           string `json:"name,omitempty"`
	AIName         string `json:"ai_name,omitempty"`
	DLCID          int    `json:"dlc_id,omitempty"`
	Color          int    `json:"color,omitempty"`
	Team           int    `json:"team,omitempty"`
	SelectedTeamID int    `json:"selected_team_id,omitempty"`
	ResolvedTeamID int    `json:"resolved_team_id,omitempty"`
	Civ            int    `json:"civ,omitempty"`
	ProfileID      int    `json:"profile_id,omitempty"`
	PlayerType     int    `json:"player_type"`
	Human          bool   `json:"human"`
	Kind           string `json:"kind"`
}

type PlayerStateSample struct {
	TimeMS            int      `json:"time_ms"`
	Time              string   `json:"time"`
	ResourceStockpile uint32   `json:"resource_stockpile"`
	UnitTypeSum       uint32   `json:"unit_type_sum"`
	ForceMetric       uint32   `json:"word_3_force_metric"`
	Word4             uint32   `json:"word_4"`
	ObjectCount       uint32   `json:"object_count"`
	PositionSum       uint32   `json:"position_sum"`
	PlayerNumber      uint32   `json:"player_number_word_8"`
	ScoreCandidate    uint32   `json:"word_9_score_candidate"`
	ObjectIDSum       uint32   `json:"object_id_sum"`
	RawWords          []uint32 `json:"raw_words_11_u32,omitempty"`
}

type PlayerStorySummary struct {
	PlayerID   int    `json:"player_id"`
	PlayerName string `json:"player_name,omitempty"`
	Kind       string `json:"kind,omitempty"`
	FirstTime  string `json:"first_time,omitempty"`
	LastTime   string `json:"last_time,omitempty"`
	Chat       int    `json:"chat"`
	Taunts     int    `json:"taunts"`
	Flares     int    `json:"flares"`
	Telemetry  int    `json:"telemetry"`
}

type PlayerTailIdent struct {
	Label            string  `json:"label"`
	Start            int     `json:"start"`
	End              int     `json:"end"`
	Bytes            int     `json:"bytes"`
	PrefixMatchVsRef int     `json:"identical_prefix_bytes_vs_reference"`
	IdenticalRatio   float64 `json:"identical_byte_ratio_vs_reference"`
	Reference        bool    `json:"is_reference,omitempty"`
}

type PlayersExpectation struct {
	MinCount     int   `json:"min_count,omitempty"`
	ExactCount   int   `json:"exact_count,omitempty"`
	RequiredIDs  []int `json:"required_ids,omitempty"`
	ForbiddenIDs []int `json:"forbidden_ids,omitempty"`
}

type PlaytestMoment struct {
	Index            int                 `json:"index"`
	RecordedCount    int                 `json:"recorded_count,omitempty"`
	TimeMS           int                 `json:"time_ms"`
	Time             string              `json:"time"`
	LastTimeMS       int                 `json:"last_time_ms,omitempty"`
	LastTime         string              `json:"last_time,omitempty"`
	Type             string              `json:"type"`
	PlayerID         int                 `json:"player_id,omitempty"`
	PlayerName       string              `json:"player_name,omitempty"`
	Text             string              `json:"text,omitempty"`
	TauntNumber      int                 `json:"taunt_number,omitempty"`
	X                float64             `json:"x,omitempty"`
	Y                float64             `json:"y,omitempty"`
	Region           string              `json:"region,omitempty"`
	Source           string              `json:"source"`
	Confidence       string              `json:"confidence"`
	ChangedPlayers   []string            `json:"changed_players,omitempty"`
	UnchangedPlayers []string            `json:"unchanged_players,omitempty"`
	Window           PlaytestStateWindow `json:"state_window"`
}

type PlaytestOptions struct {
	// Window is the number of checksum samples before and after a feedback
	// moment to include. Values <= 0 use the default.
	Window           int  `json:"window,omitempty"`
	IncludeComputers bool `json:"include_computers,omitempty"`
}

type PlaytestPlayer struct {
	PlayerID int    `json:"player_id"`
	Label    string `json:"label"`
	Name     string `json:"name,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Human    bool   `json:"human"`
}

type PlaytestReport struct {
	Version      string           `json:"version"`
	Path         string           `json:"path,omitempty"`
	Method       string           `json:"method"`
	Verification string           `json:"verification"`
	Session      PlaytestSession  `json:"session"`
	Summary      PlaytestSummary  `json:"summary"`
	Moments      []PlaytestMoment `json:"moments,omitempty"`
	IssueCards   []IssueCard      `json:"issue_cards,omitempty"`
	Honesty      []string         `json:"honesty"`
	Warnings     []string         `json:"warnings,omitempty"`
}

func BuildPlaytestReport(path string, opts PlaytestOptions) (*PlaytestReport, error)

type PlaytestSession struct {
	ScenarioName       string           `json:"scenario_name,omitempty"`
	DurationMS         int              `json:"duration_ms,omitempty"`
	Duration           string           `json:"duration,omitempty"`
	TriggerGraphSHA256 string           `json:"trigger_graph_sha256,omitempty"`
	TriggerGraphOK     bool             `json:"trigger_graph_ok"`
	TriggerGraphMethod string           `json:"trigger_graph_method,omitempty"`
	DataSet            DataSetIdentity  `json:"data_set_identity"`
	HumanCount         int              `json:"human_count"`
	ComputerCount      int              `json:"computer_count"`
	Players            []PlaytestPlayer `json:"players,omitempty"`
}

type PlaytestStateChange struct {
	PlayerID         int    `json:"player_id"`
	PlayerLabel      string `json:"player_label"`
	PlayerName       string `json:"player_name,omitempty"`
	Changed          bool   `json:"changed"`
	Kind             string `json:"kind"`
	ObjectCountDelta int64  `json:"object_count_delta,omitempty"`
	UnitTypeSumDelta int64  `json:"unit_type_sum_delta,omitempty"`
	UnitID           int    `json:"unit_id,omitempty"`
	UnitName         string `json:"unit_name,omitempty"`
	Confidence       string `json:"confidence,omitempty"`
}

type PlaytestStateInterval struct {
	FromTimeMS int                   `json:"from_time_ms"`
	FromTime   string                `json:"from_time"`
	ToTimeMS   int                   `json:"to_time_ms"`
	ToTime     string                `json:"to_time"`
	Players    []PlaytestStateChange `json:"players"`
}

type PlaytestStateWindow struct {
	SamplesBefore      int                     `json:"samples_before"`
	SamplesAfter       int                     `json:"samples_after"`
	FocusIntervalIndex int                     `json:"focus_interval_index,omitempty"`
	StartTimeMS        int                     `json:"start_time_ms,omitempty"`
	StartTime          string                  `json:"start_time,omitempty"`
	EndTimeMS          int                     `json:"end_time_ms,omitempty"`
	EndTime            string                  `json:"end_time,omitempty"`
	Intervals          []PlaytestStateInterval `json:"intervals,omitempty"`
}

type PlaytestSummary struct {
	FeedbackMoments      int `json:"feedback_moments"`
	RawFeedbackRows      int `json:"raw_feedback_rows,omitempty"`
	BacklogRowsExcluded  int `json:"backlog_rows_excluded,omitempty"`
	ChecksumSamples      int `json:"checksum_samples"`
	StateDeltas          int `json:"state_deltas"`
	WindowSamplesEachWay int `json:"window_samples_each_way"`
}

type PostgameCorpusReport struct {
	Folder       string                `json:"folder"`
	Method       string                `json:"method"`
	Verification string                `json:"verification"`
	Scanned      int                   `json:"replays_scanned"`
	Failures     int                   `json:"parse_failures"`
	Rows         []PostgameCorpusRow   `json:"rows,omitempty"`
	Summary      PostgameCorpusSummary `json:"summary"`
	Warnings     []string              `json:"warnings,omitempty"`
}

func BuildPostgameCorpus(folder string) (*PostgameCorpusReport, error)

type PostgameCorpusRow struct {
	Path              string `json:"path"`
	Op6Seen           bool   `json:"op6_seen"`
	Op6TailBytes      int    `json:"op6_tail_bytes,omitempty"`
	Op6Blocks         int    `json:"op6_blocks,omitempty"`
	WorldTimeBlock    bool   `json:"world_time_block"`
	LeaderboardBlock  bool   `json:"leaderboard_block"`
	Action255Payloads int    `json:"action255_payloads"`
	UnparsedOp6Tail   bool   `json:"unparsed_op6_tail"`
	Candidate         bool   `json:"achievements_candidate"`
	Confidence        string `json:"confidence"`
	Error             string `json:"error,omitempty"`
}

type PostgameCorpusSummary struct {
	Op6Present        int `json:"op6_present"`
	Op6MetadataOnly   int `json:"op6_metadata_only"`
	Op6UnparsedTails  int `json:"op6_unparsed_tails"`
	Action255Carriers int `json:"action255_carriers"`
	Candidates        int `json:"achievements_candidates"`
}

type PostgamePlayerResult struct {
	PlayerID       int    `json:"player_id"`
	Label          string `json:"label"`
	Name           string `json:"name,omitempty"`
	Winner         *bool  `json:"winner,omitempty"`
	Score          *int   `json:"score,omitempty"`
	UnitsKilled    *int   `json:"units_killed,omitempty"`
	UnitsLost      *int   `json:"units_lost,omitempty"`
	BuildingsRazed *int   `json:"buildings_razed,omitempty"`
	BuildingsLost  *int   `json:"buildings_lost,omitempty"`
	FeudalTimeMS   *int   `json:"feudal_time_ms,omitempty"`
	CastleTimeMS   *int   `json:"castle_time_ms,omitempty"`
	ImperialTimeMS *int   `json:"imperial_time_ms,omitempty"`
	Confidence     string `json:"confidence"`
}

type PostgameReport struct {
	Path         string                 `json:"path,omitempty"`
	Method       string                 `json:"method"`
	Verification string                 `json:"verification"`
	Summary      PostgameSummary        `json:"summary"`
	DE           *DEPostgame            `json:"de_postgame,omitempty"`
	Action255    []Action255Postgame    `json:"action_255_postgame,omitempty"`
	Players      []PostgamePlayerResult `json:"players,omitempty"`
	Warnings     []string               `json:"warnings,omitempty"`
}

func BuildPostgame(path string) (*PostgameReport, error)

type PostgameSummary struct {
	Op6Seen             bool `json:"op6_seen"`
	Op6TailBytes        int  `json:"op6_tail_bytes,omitempty"`
	Op6Blocks           int  `json:"op6_blocks,omitempty"`
	Action255Seen       bool `json:"action255_seen"`
	Action255Payloads   int  `json:"action255_payloads,omitempty"`
	HasLeaderboardBlock bool `json:"has_leaderboard_block"`
	HasWorldTimeBlock   bool `json:"has_world_time_block"`
	HasPlayerKills      bool `json:"has_player_kills"`
}

type ProfileCoverage struct {
	ReplayActions      int     `json:"replay_actions"`
	DecodedActions     int     `json:"decoded_actions"`
	UndecodedActions   int     `json:"undecoded_actions"`
	DecodedPercent     float64 `json:"decoded_percent"`
	CoordinateActions  int     `json:"coordinate_actions"`
	ViewlockEvents     int     `json:"viewlock_events"`
	FeedbackEvents     int     `json:"feedback_events"`
	PhaseTagged        int     `json:"phase_tagged_feedback"`
	PhaseHeuristicNote string  `json:"phase_heuristic_note,omitempty"`
}

type ProfileEventRow struct {
	Index       int     `json:"index"`
	Type        string  `json:"type"`
	TimeMS      int     `json:"time_ms"`
	Time        string  `json:"time"`
	Phase       string  `json:"phase,omitempty"`
	PlayerID    int     `json:"player_id,omitempty"`
	PlayerName  string  `json:"player_name,omitempty"`
	ActionID    int     `json:"action_id,omitempty"`
	ActionName  string  `json:"action_name,omitempty"`
	X           float64 `json:"x,omitempty"`
	Y           float64 `json:"y,omitempty"`
	Region      string  `json:"region,omitempty"`
	Text        string  `json:"text,omitempty"`
	TauntNumber int     `json:"taunt_number,omitempty"`
	Confidence  string  `json:"confidence"`
}

type ProfileExpectation struct {
	MinReplayActions  int                       `json:"min_replay_actions,omitempty"`
	MinDecodedPercent float64                   `json:"min_decoded_percent,omitempty"`
	Players           []PlayerActionExpectation `json:"players,omitempty"`
}

type ProfileFeedbackCount struct {
	Chat         int            `json:"chat"`
	Taunts       int            `json:"taunts"`
	Flares       int            `json:"flares"`
	Telemetry    int            `json:"telemetry"`
	ByPhase      map[string]int `json:"by_phase,omitempty"`
	ChatByPhase  map[string]int `json:"chat_by_phase,omitempty"`
	TauntByPhase map[string]int `json:"taunt_by_phase,omitempty"`
}

type ProfileNotable struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

type RegionSummary struct {
	Name   string `json:"name"`
	Events int    `json:"events"`
	Flares int    `json:"flares"`
}

type RenderExpectation struct {
	ID          string `json:"id,omitempty"`
	Description string `json:"description"`
}

type ReplayCache struct {
	Dir      string
	Source   string
	Manifest CacheManifest
}

func OpenReplayCache(path string) (*ReplayCache, error)
    OpenReplayCache hashes with a fixed buffer. A completed manifest is the
    commit marker; interrupted builds never become visible as usable cache
    entries.

func (c *ReplayCache) HeaderReader() (*os.File, error)
    HeaderReader transfers ownership of a read-only disk handle to the caller.

func (c *ReplayCache) LobbyChat(ids map[string]int) ([]ChatLine, error)
    LobbyChat scans the disk header in overlapping bounded windows. Offsets are
    absolute and deduplicated across overlaps. No full-header string copy.

func (c *ReplayCache) Metadata() (*File, error)
    Metadata inspects bounded prefix/tail windows. Full snapshot bytes remain on
    disk. An unavailable graph is explicit, never a guessed successful parse.

func (c *ReplayCache) WalkOps(visit func(CachedOp) error) error

type ReplayCorpusReport struct {
	Folder       string               `json:"folder"`
	Method       string               `json:"method"`
	Verification string               `json:"verification"`
	Summary      ReplayCorpusSummary  `json:"summary"`
	Rows         []ReplayCorpusRow    `json:"rows"`
	Unknowns     []CorpusUnknownGroup `json:"unknown_actions,omitempty"`
	Warnings     []string             `json:"warnings,omitempty"`
}

func BuildReplayCorpus(folder string, opts CorpusOptions) (*ReplayCorpusReport, error)

type ReplayCorpusRow struct {
	Path                 string      `json:"path"`
	OK                   bool        `json:"ok"`
	Error                string      `json:"error,omitempty"`
	FileBytes            int         `json:"file_bytes,omitempty"`
	GameVersion          string      `json:"game_version,omitempty"`
	SaveVersion          float64     `json:"save_version,omitempty"`
	LogVersion           uint32      `json:"log_version,omitempty"`
	ReplayKind           string      `json:"replay_kind,omitempty"`
	ScenarioIdentity     string      `json:"scenario_identity,omitempty"`
	ScenarioIdentityTier string      `json:"scenario_identity_tier,omitempty"`
	TriggerCount         int         `json:"trigger_count,omitempty"`
	EffectCount          int         `json:"effect_count,omitempty"`
	ConditionCount       int         `json:"condition_count,omitempty"`
	MessageCount         int         `json:"message_count,omitempty"`
	MapWidth             int         `json:"map_width,omitempty"`
	MapHeight            int         `json:"map_height,omitempty"`
	DataSetStatus        string      `json:"data_set_status,omitempty"`
	ActiveDataSet        string      `json:"active_data_set,omitempty"`
	PlayerCount          int         `json:"player_count,omitempty"`
	HumanCount           int         `json:"human_count,omitempty"`
	AICount              int         `json:"ai_count,omitempty"`
	PlayerNames          []string    `json:"player_names,omitempty"`
	DurationMS           int         `json:"duration_ms,omitempty"`
	Duration             string      `json:"duration,omitempty"`
	Actions              int         `json:"actions,omitempty"`
	UntypedActions       int         `json:"untyped_actions,omitempty"`
	UnknownOps           int         `json:"unknown_ops,omitempty"`
	Chat                 int         `json:"chat,omitempty"`
	Taunts               int         `json:"taunts,omitempty"`
	Flares               int         `json:"flares,omitempty"`
	Resigns              int         `json:"resigns,omitempty"`
	Postgames            int         `json:"postgames,omitempty"`
	HeaderBytes          int         `json:"header_bytes,omitempty"`
	BodyBytes            int         `json:"body_bytes,omitempty"`
	HeaderOpaqueBytes    int         `json:"header_opaque_bytes,omitempty"`
	BodyOpaqueBytes      int         `json:"body_opaque_bytes,omitempty"`
	HeaderDecodedPercent float64     `json:"header_decoded_percent,omitempty"`
	BodyDecodedPercent   float64     `json:"body_decoded_percent,omitempty"`
	BodyOps              map[int]int `json:"body_ops,omitempty"`
	ActionIDs            map[int]int `json:"action_ids,omitempty"`
	UnknownActionIDs     map[int]int `json:"unknown_action_ids,omitempty"`
	PostgameShape        string      `json:"postgame_shape,omitempty"`
	PostgameVerification string      `json:"postgame_verification,omitempty"`
	Warnings             []string    `json:"warnings,omitempty"`
}

type ReplayCorpusSummary struct {
	Scanned                 int            `json:"scanned"`
	Failures                int            `json:"failures"`
	ScenarioReplays         int            `json:"scenario_replays"`
	NonScenarioReplays      int            `json:"non_scenario_replays"`
	TriggerGraphs           int            `json:"trigger_graphs"`
	FallbackFingerprints    int            `json:"fallback_fingerprints"`
	Players                 int            `json:"players"`
	Humans                  int            `json:"humans"`
	AIs                     int            `json:"ais"`
	TotalActions            int            `json:"total_actions"`
	UntypedActions          int            `json:"untyped_actions"`
	UnknownOps              int            `json:"unknown_ops"`
	Chat                    int            `json:"chat"`
	Taunts                  int            `json:"taunts"`
	Flares                  int            `json:"flares"`
	Resigns                 int            `json:"resigns"`
	Postgames               int            `json:"postgames"`
	BodyOps                 map[int]int    `json:"body_ops,omitempty"`
	ActionIDs               map[int]int    `json:"action_ids,omitempty"`
	HeaderBytes             int            `json:"header_bytes"`
	BodyBytes               int            `json:"body_bytes"`
	HeaderOpaqueBytes       int            `json:"header_opaque_bytes"`
	BodyOpaqueBytes         int            `json:"body_opaque_bytes"`
	HeaderDecodedPercentMin float64        `json:"header_decoded_percent_min,omitempty"`
	HeaderDecodedPercentAvg float64        `json:"header_decoded_percent_avg,omitempty"`
	BodyDecodedPercentMin   float64        `json:"body_decoded_percent_min,omitempty"`
	BodyDecodedPercentAvg   float64        `json:"body_decoded_percent_avg,omitempty"`
	PostgameShapes          map[string]int `json:"postgame_shapes,omitempty"`
}

type ReplayEvent struct {
	Index          int               `json:"index"`
	Type           string            `json:"type"`
	TimeMS         int               `json:"time_ms,omitempty"`
	Time           string            `json:"time,omitempty"`
	PlayerID       int               `json:"player_id,omitempty"`
	PlayerName     string            `json:"player_name,omitempty"`
	Channel        int               `json:"channel,omitempty"`
	ChannelName    string            `json:"channel_name,omitempty"`
	Text           string            `json:"text,omitempty"`
	Phase          string            `json:"phase,omitempty"`
	TauntNumber    int               `json:"taunt_number,omitempty"`
	TauntText      string            `json:"taunt_text,omitempty"`
	DestinationMap int               `json:"destination_map,omitempty"`
	MessageAGP     string            `json:"message_agp,omitempty"`
	X              float64           `json:"x,omitempty"`
	Y              float64           `json:"y,omitempty"`
	XEnd           float64           `json:"x_end,omitempty"`
	YEnd           float64           `json:"y_end,omitempty"`
	Region         string            `json:"region,omitempty"`
	Targets        []int             `json:"targets,omitempty"`
	OperationID    int               `json:"operation_id,omitempty"`
	ReplayAction   bool              `json:"replay_action,omitempty"`
	ActionID       int               `json:"action_id,omitempty"`
	ActionName     string            `json:"action_name,omitempty"`
	SourceOffset   int               `json:"source_offset,omitempty"`
	PayloadBytes   int               `json:"payload_bytes,omitempty"`
	RawHex         string            `json:"raw_hex,omitempty"`
	RawText        string            `json:"raw_text,omitempty"`
	Source         string            `json:"source"`
	Confidence     string            `json:"confidence"`
	Telemetry      *TelemetryEvent   `json:"telemetry,omitempty"`
	ObjectIDs      []int             `json:"object_ids,omitempty"`
	TargetID       int               `json:"target_id,omitempty"`
	CommandID      int               `json:"command_id,omitempty"`
	UnitID         int               `json:"unit_id,omitempty"`
	TechnologyID   int               `json:"technology_id,omitempty"`
	Amount         int               `json:"amount,omitempty"`
	BuildingID     int               `json:"building_id,omitempty"`
	TargetObject   *ObjectReference  `json:"target_object,omitempty"`
	BuildingObject *ObjectReference  `json:"building_object,omitempty"`
	ObjectRefs     []ObjectReference `json:"object_refs,omitempty"`
	ModeID         int               `json:"mode_id,omitempty"`
	StanceID       int               `json:"stance_id,omitempty"`
	FormationID    int               `json:"formation_id,omitempty"`
	OrderID        int               `json:"order_id,omitempty"`
	SlotID         int               `json:"slot_id,omitempty"`
	Sequence       int               `json:"sequence,omitempty"`
}

func DecodeActionEvent(actionID int, payload []byte, timeMS int, offset int, includeRaw bool) (ReplayEvent, bool)

func ExtractJSONScanEvents(body []byte, opts EventOptions) ([]ReplayEvent, map[int]string, []string)

type ReplayRegion struct {
	Space      string         `json:"space"`
	Name       string         `json:"name"`
	Start      int            `json:"start"`
	End        int            `json:"end"`
	Bytes      int            `json:"bytes"`
	Status     string         `json:"status"`
	Confidence string         `json:"confidence"`
	Details    map[string]any `json:"details,omitempty"`
}

type ResignEvent struct {
	TimeMS   int    `json:"time_ms"`
	Time     string `json:"time"`
	PlayerID int    `json:"player_id"`
	Sequence int    `json:"sequence,omitempty"`
}

type ResultExpectation struct {
	Winners []int `json:"winners,omitempty"`
	Losers  []int `json:"losers,omitempty"`
}

type ScenarioExpectation struct {
	Tier        string `json:"tier,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

type ScenarioMetadata struct {
	Name              string `json:"name,omitempty"`
	Description       string `json:"description,omitempty"`
	NameOffset        int    `json:"name_offset,omitempty"`
	DescriptionOffset int    `json:"description_offset,omitempty"`
	Source            string `json:"source"`
	Confidence        string `json:"confidence"`
	Error             string `json:"error,omitempty"`
}

type SidecarExpectation struct {
	ResourceStockpile      int  `json:"resource_stockpile"`
	UnitTypeSum            int  `json:"unit_type_sum"`
	ObjectCount            int  `json:"object_count"`
	ResourceStockpileKnown bool `json:"resource_stockpile_known"`
	UnitTypeSumKnown       bool `json:"unit_type_sum_known"`
	ObjectCountKnown       bool `json:"object_count_known"`
}

type SidecarKnownWordCheck struct {
	WordIndex int    `json:"word_index"`
	Name      string `json:"name"`
	Expected  int    `json:"expected"`
	Actual    int    `json:"actual"`
	Match     bool   `json:"match"`
}

type SidecarStatePhase struct {
	Index    int                `json:"index"`
	PhaseID  int                `json:"phase_id"`
	Label    string             `json:"label,omitempty"`
	TimerS   int                `json:"ledger_timer_s,omitempty"`
	XSTimeS  int                `json:"xs_time_s"`
	TargetMS int                `json:"target_ms"`
	Food     float64            `json:"food"`
	Wood     float64            `json:"wood"`
	Gold     float64            `json:"gold"`
	Stone    float64            `json:"stone"`
	Attr20   float64            `json:"attr20"`
	Attr33   float64            `json:"attr33"`
	Attr220  float64            `json:"attr220"`
	Counts   map[string]int     `json:"counts"`
	Expected SidecarExpectation `json:"expected"`
}

type SidecarStateVector struct {
	Magic       string              `json:"magic,omitempty"`
	Version     int                 `json:"version,omitempty"`
	Label       string              `json:"label,omitempty"`
	RowsWritten int                 `json:"rows_written,omitempty"`
	AppendProbe int                 `json:"append_probe,omitempty"`
	Phases      []SidecarStatePhase `json:"phases,omitempty"`
}

type SidecarSyncCorrelation struct {
	Phase             SidecarStatePhase          `json:"phase"`
	Verdict           string                     `json:"verdict"`
	Confidence        string                     `json:"confidence"`
	Sample            *SidecarSyncSample         `json:"sample,omitempty"`
	DeltaFromPrevious []int64                    `json:"delta_words_from_previous_sample,omitempty"`
	KnownWordChecks   []SidecarKnownWordCheck    `json:"known_word_checks,omitempty"`
	CarrierMatch      *bool                      `json:"carrier_word_1_match,omitempty"`
	UnitTypeMatch     *bool                      `json:"unit_type_word_2_match,omitempty"`
	ObjectCountMatch  *bool                      `json:"object_count_word_6_match,omitempty"`
	AttributeMirrors  []AttributeMirrorCandidate `json:"attribute_mirrors,omitempty"`
	Notes             []string                   `json:"notes,omitempty"`
}

type SidecarSyncOptions struct {
	SidecarPath string
	LedgerPath  string
	Schema      string
	PlayerID    int
	WindowMS    int
}

type SidecarSyncReport struct {
	Path          string                           `json:"path,omitempty"`
	SidecarPath   string                           `json:"sidecar_path,omitempty"`
	LedgerPath    string                           `json:"ledger_path,omitempty"`
	Schema        string                           `json:"schema,omitempty"`
	Method        string                           `json:"method"`
	Verification  string                           `json:"verification"`
	Summary       SidecarSyncSummary               `json:"summary"`
	Sidecar       SidecarStateVector               `json:"sidecar"`
	Correlations  []SidecarSyncCorrelation         `json:"correlations,omitempty"`
	Warnings      []string                         `json:"warnings,omitempty"`
	Decode        *xsauthor.DataDecodeReport       `json:"xsdat_decode,omitempty"`
	SchemaDecode  *xsauthor.DataSchemaDecodeReport `json:"xsdat_schema_decode,omitempty"`
	WordSemantics map[string]string                `json:"checksum_word_semantics,omitempty"`
}

func BuildSidecarSyncReport(path string, opts SidecarSyncOptions) (*SidecarSyncReport, error)

type SidecarSyncSample struct {
	TimeMS int      `json:"time_ms"`
	Time   string   `json:"time"`
	LagMS  int      `json:"lag_ms"`
	Words  []uint32 `json:"words_u32"`
}

type SidecarSyncSummary struct {
	PhaseCount          int  `json:"phase_count"`
	ChecksumSamples     int  `json:"checksum_samples"`
	SidecarDecodeOK     bool `json:"sidecar_decode_ok"`
	LedgerAssertions    int  `json:"ledger_assertions"`
	LedgerPassed        int  `json:"ledger_passed"`
	LedgerFailed        int  `json:"ledger_failed"`
	LedgerUnknown       int  `json:"ledger_unknown"`
	MatchedPhases       int  `json:"matched_phases"`
	MissingSamples      int  `json:"missing_samples"`
	KnownWordChecks     int  `json:"known_word_checks"`
	KnownWordMatches    int  `json:"known_word_matches"`
	KnownWordMismatches int  `json:"known_word_mismatches"`
	CarrierMatches      int  `json:"carrier_matches"`
	UnitTypeMatches     int  `json:"unit_type_sum_matches"`
	ObjectCountMatches  int  `json:"object_count_matches"`
	AttrMirrorHits      int  `json:"attribute_mirror_hits"`
}

type SpanStats struct {
	Entropy        float64 `json:"entropy"`
	DistinctBytes  int     `json:"distinct_bytes"`
	ZeroBytes      int     `json:"zero_bytes"`
	FFBytes        int     `json:"ff_bytes"`
	PrintableBytes int     `json:"printable_bytes"`
}

type SpatialCell struct {
	Cell  string  `json:"cell"`
	MinX  float64 `json:"min_x"`
	MinY  float64 `json:"min_y"`
	Count int     `json:"count"`
}

type SpatialProfile struct {
	Commands           int             `json:"commands"`
	CellsVisited       int             `json:"cells_visited"`
	CellSize           float64         `json:"cell_size"`
	CentroidSwitches   int             `json:"centroid_switches"`
	TopCells           []SpatialCell   `json:"top_cells,omitempty"`
	TopRegions         []RegionSummary `json:"top_regions,omitempty"`
	CoordinateCoverage float64         `json:"coordinate_coverage_percent,omitempty"`
}

type SpawnCatalogReport struct {
	Path         string               `json:"path,omitempty"`
	Method       string               `json:"method"`
	Verification string               `json:"verification"`
	Summary      SpawnCatalogSummary  `json:"summary"`
	Players      []SpawnPlayerSummary `json:"players,omitempty"`
	Spawns       []SpawnRecipe        `json:"spawns,omitempty"`
	Warnings     []string             `json:"warnings,omitempty"`
}

func BuildSpawnCatalog(path string) (*SpawnCatalogReport, error)

type SpawnCatalogSummary struct {
	CreateObjectEffects int `json:"create_object_effects"`
	PlausibleSpawns     int `json:"plausible_spawns"`
	Players             int `json:"players"`
	UnitTypes           int `json:"unit_types"`
}

type SpawnPlayerSummary struct {
	PlayerID int         `json:"player_id"`
	Label    string      `json:"label"`
	Spawns   int         `json:"spawns"`
	Units    map[int]int `json:"units,omitempty"`
}

type SpawnRecipe struct {
	TriggerIndex int    `json:"trigger_index"`
	TriggerName  string `json:"trigger_name,omitempty"`
	EffectIndex  int    `json:"effect_index"`
	UnitID       int    `json:"unit_id"`
	UnitName     string `json:"unit_name,omitempty"`
	TargetPlayer int    `json:"target_player"`
	X            int    `json:"x"`
	Y            int    `json:"y"`
	Confidence   string `json:"confidence"`
}

type StoryChapter struct {
	Name       string        `json:"name"`
	StartMS    int           `json:"start_ms"`
	EndMS      int           `json:"end_ms"`
	Start      string        `json:"start"`
	End        string        `json:"end"`
	Counts     ChapterCounts `json:"counts"`
	Highlights []StoryLine   `json:"highlights,omitempty"`
}

type StoryClaim struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Source     string `json:"source"`
	Confidence string `json:"confidence"`
	Detail     string `json:"detail,omitempty"`
}

type StoryIdentity struct {
	Tier           string `json:"tier,omitempty"`
	SHA256         string `json:"sha256,omitempty"`
	TriggerGraphOK bool   `json:"trigger_graph_ok"`
	Method         string `json:"method,omitempty"`
}

type StoryLine struct {
	Time       string `json:"time,omitempty"`
	TimeMS     int    `json:"time_ms,omitempty"`
	PlayerID   int    `json:"player_id,omitempty"`
	PlayerName string `json:"player_name,omitempty"`
	Region     string `json:"region,omitempty"`
	Type       string `json:"type"`
	Text       string `json:"text"`
}

type StoryMoment struct {
	Kind       string `json:"kind"`
	Time       string `json:"time,omitempty"`
	TimeMS     int    `json:"time_ms,omitempty"`
	PlayerID   int    `json:"player_id,omitempty"`
	PlayerName string `json:"player_name,omitempty"`
	Detail     string `json:"detail"`
	Source     string `json:"source"`
	Confidence string `json:"confidence"`
}

type StoryOptions struct {
	ContextPath string
}

type StoryReport struct {
	Path             string               `json:"path,omitempty"`
	ContextName      string               `json:"context_name,omitempty"`
	Identity         StoryIdentity        `json:"identity"`
	DataSet          DataSetIdentity      `json:"data_set_identity"`
	Players          []FeedbackPlayer     `json:"players,omitempty"`
	PlayerSummary    []PlayerStorySummary `json:"player_summary,omitempty"`
	Brief            []string             `json:"brief"`
	Timeline         []StoryLine          `json:"timeline,omitempty"`
	Feedback         []StoryLine          `json:"feedback,omitempty"`
	Telemetry        []StoryLine          `json:"telemetry,omitempty"`
	Chapters         []StoryChapter       `json:"chapters,omitempty"`
	Moments          []StoryMoment        `json:"moments,omitempty"`
	Regions          []RegionSummary      `json:"regions,omitempty"`
	FeedbackClusters []FeedbackCluster    `json:"feedback_clusters,omitempty"`
	EventCounts      EventCounts          `json:"event_counts"`
	Result           MatchResult          `json:"result"`
	Verification     string               `json:"verification"`
	Claims           []StoryClaim         `json:"claims,omitempty"`
	Missing          []string             `json:"missing,omitempty"`
	Warnings         []string             `json:"warnings,omitempty"`
}

func BuildStory(path string, opts StoryOptions) (*StoryReport, error)

type SummaryOptions struct {
	IncludeMapTiles bool
}

type SummaryPlayer struct {
	Slot           int    `json:"slot"`
	PlayerID       int    `json:"player_id,omitempty"`
	Name           string `json:"name,omitempty"`
	AIName         string `json:"ai_name,omitempty"`
	Kind           string `json:"kind"`
	Human          bool   `json:"human"`
	Civ            int    `json:"civ,omitempty"`
	CivName        string `json:"civ_name,omitempty"`
	Color          int    `json:"color,omitempty"`
	Team           int    `json:"team,omitempty"`
	ProfileID      int    `json:"profile_id,omitempty"`
	PlayerType     int    `json:"player_type"`
	Rating         *int   `json:"rating,omitempty"`
	Winner         *bool  `json:"winner,omitempty"`
	Resigned       bool   `json:"resigned,omitempty"`
	SelectedTeamID int    `json:"selected_team_id,omitempty"`
	ResolvedTeamID int    `json:"resolved_team_id,omitempty"`
}

type SummaryReport struct {
	Version       string            `json:"version"`
	Path          string            `json:"path,omitempty"`
	RecordSHA256  string            `json:"record_sha256,omitempty"`
	GameVersion   string            `json:"game_version"`
	SaveVersion   float64           `json:"save_version"`
	LogVersion    uint32            `json:"log_version,omitempty"`
	HeaderLength  int               `json:"header_length"`
	InflatedBytes int               `json:"inflated_header_bytes"`
	DurationMS    int               `json:"duration_ms,omitempty"`
	Duration      string            `json:"duration,omitempty"`
	Map           *MapInfo          `json:"map,omitempty"`
	DataSet       DataSetIdentity   `json:"data_set_identity"`
	LobbySettings *LobbySettings    `json:"lobby_settings,omitempty"`
	ScenarioName  string            `json:"scenario_name,omitempty"`
	ScenarioMeta  *ScenarioMetadata `json:"scenario_metadata,omitempty"`
	HeaderMeta    *HeaderMetadata   `json:"header_metadata,omitempty"`
	Players       []SummaryPlayer   `json:"players,omitempty"`
	Result        MatchResult       `json:"result"`
	Postgame      *PostgameSummary  `json:"postgame_summary,omitempty"`
	Warnings      []string          `json:"warnings,omitempty"`
}

func BuildSummary(path string) (*SummaryReport, error)

func BuildSummaryWithOptions(path string, opts SummaryOptions) (*SummaryReport, error)

type SyncEvent struct {
	TimeMS       int        `json:"time_ms"`
	Time         string     `json:"time"`
	SourceOffset int        `json:"source_offset"`
	DeltaMS      int        `json:"delta_ms"`
	Form         string     `json:"form"`
	ProbeHex     string     `json:"probe_hex,omitempty"`
	Matrix       [][]uint32 `json:"matrix_8x11_u32,omitempty"`
	TrailerU32   *uint32    `json:"trailer_u32,omitempty"`
	PayloadHex   string     `json:"payload_hex,omitempty"`
}

type SyncLogChecksumEstimate struct {
	ResourceStockpileSum float64 `json:"resource_stockpile_word_1_candidate,omitempty"`
	ObjectCount          int     `json:"object_count_word_6_candidate"`
	UnitTypeSum          int64   `json:"unit_type_sum_word_2_candidate"`
	ObjectIDSum          int64   `json:"object_id_sum_word_10_candidate"`
	PositiveObjectIDs    int64   `json:"positive_object_id_sum_candidate"`
	PositionX100         int64   `json:"position_x_100_sum_candidate"`
	PositionY100         int64   `json:"position_y_100_sum_candidate"`
	PositionXY100        int64   `json:"position_xy_100_sum_candidate"`
	PositionXYZ100       int64   `json:"position_xyz_100_sum_candidate"`
	RetargetTimerSum     int64   `json:"retarget_timer_sum_candidate"`
	StateSum             int64   `json:"state_sum_candidate"`
	CarrySum             int64   `json:"carry_sum_candidate"`
	ActionTypeSum        int64   `json:"action_type_sum_candidate"`
	ActionTypeNonNegSum  int64   `json:"action_type_non_negative_sum_candidate"`
	ActionStateSum       int64   `json:"action_state_sum_candidate"`
	HP100Sum             int64   `json:"hp_100_sum_candidate"`
}

type SyncLogCorpseCarry struct {
	PlayerID       int     `json:"player_id"`
	ObjectID       int     `json:"object_id"`
	UnitID         int     `json:"unit_id"`
	UnitName       string  `json:"unit_name"`
	Samples        int     `json:"samples"`
	FirstTimeMS    int     `json:"first_time_ms"`
	FirstTime      string  `json:"first_time"`
	LastTimeMS     int     `json:"last_time_ms"`
	LastTime       string  `json:"last_time"`
	FirstCarry     int     `json:"first_carry"`
	LastCarry      int     `json:"last_carry"`
	CarryDrop      int     `json:"carry_drop"`
	ObservedRatePS float64 `json:"observed_rate_per_second,omitempty"`
}

type SyncLogCorpseDecay struct {
	UnitID              int     `json:"unit_id"`
	UnitName            string  `json:"unit_name"`
	Tracks              int     `json:"tracks"`
	MovingTracks        int     `json:"moving_tracks"`
	Samples             int     `json:"samples"`
	FirstCarryMin       int     `json:"first_carry_min,omitempty"`
	FirstCarryMax       int     `json:"first_carry_max,omitempty"`
	LastCarryMin        int     `json:"last_carry_min,omitempty"`
	LastCarryMax        int     `json:"last_carry_max,omitempty"`
	CarryDropTotal      int     `json:"carry_drop_total"`
	DurationMSTotal     int     `json:"duration_ms_total"`
	ObservedRatePS      float64 `json:"observed_rate_per_second,omitempty"`
	ObservedRateMinPS   float64 `json:"observed_rate_min_per_second,omitempty"`
	ObservedRateMaxPS   float64 `json:"observed_rate_max_per_second,omitempty"`
	EstimatedLifetimeMS int     `json:"estimated_lifetime_ms,omitempty"`
	EstimatedLifetime   string  `json:"estimated_lifetime,omitempty"`
	Confidence          string  `json:"confidence"`
}

type SyncLogObjectCensus struct {
	Name  string `json:"name"`
	DBID  int    `json:"dbid"`
	Count int    `json:"count"`
}

type SyncLogOptions struct {
	ReplayPath string
	Limit      int
}

type SyncLogPlayerAttr struct {
	Index int     `json:"index"`
	Value float64 `json:"value"`
}

type SyncLogPlayerSnapshot struct {
	PlayerID        int                     `json:"player_id"`
	PlayerLabel     string                  `json:"player_label"`
	Kind            string                  `json:"kind,omitempty"`
	DeclaredAttrs   int                     `json:"declared_attributes,omitempty"`
	Attributes      []SyncLogPlayerAttr     `json:"attributes,omitempty"`
	DeclaredObjects int                     `json:"declared_objects,omitempty"`
	ObjectRows      int                     `json:"object_rows"`
	Checksum        SyncLogChecksumEstimate `json:"checksum_estimate"`
	// Has unexported fields.
}

type SyncLogRNGSite struct {
	Operation string `json:"operation"`
	Line      int    `json:"line"`
	Count     int    `json:"count"`
}

type SyncLogRNGStream struct {
	Stream   string           `json:"stream"`
	SetSeeds int              `json:"set_seeds"`
	Rands    int              `json:"rands"`
	Events   int              `json:"events"`
	Sites    []SyncLogRNGSite `json:"sites,omitempty"`
}

type SyncLogReplayApprox struct {
	ReplayTimeMS   int                          `json:"replay_time_ms"`
	ReplayTime     string                       `json:"replay_time"`
	LogWorldTimeMS int                          `json:"log_world_time_ms"`
	LogWorldTime   string                       `json:"log_world_time"`
	GapMS          int                          `json:"gap_ms"`
	Gap            string                       `json:"gap"`
	Players        []SyncLogReplayPlayerCompare `json:"players,omitempty"`
	Confidence     string                       `json:"confidence"`
}

type SyncLogReplayCompare struct {
	ReplayPath              string               `json:"replay_path"`
	ChecksumSamples         int                  `json:"checksum_samples"`
	ReplayFirstTimeMS       int                  `json:"replay_first_time_ms,omitempty"`
	ReplayLastTimeMS        int                  `json:"replay_last_time_ms,omitempty"`
	LogFirstWorldTimeMS     int                  `json:"log_first_world_time_ms,omitempty"`
	LogLastWorldTimeMS      int                  `json:"log_last_world_time_ms,omitempty"`
	OverlappingSamples      int                  `json:"overlapping_samples"`
	Overlap                 bool                 `json:"overlap"`
	ExactWorldTimeMatches   []SyncLogReplayMatch `json:"exact_world_time_matches,omitempty"`
	NearestBeforeLogStartMS int                  `json:"nearest_before_log_start_ms,omitempty"`
	NearestBeforeLogStart   string               `json:"nearest_before_log_start,omitempty"`
	NearestAfterLogEndMS    int                  `json:"nearest_after_log_end_ms,omitempty"`
	NearestAfterLogEnd      string               `json:"nearest_after_log_end,omitempty"`
	NearestBeforeToFirstLog *SyncLogReplayApprox `json:"nearest_before_to_first_log,omitempty"`
	Warning                 string               `json:"warning,omitempty"`
}

type SyncLogReplayMatch struct {
	TimeMS  int                          `json:"time_ms"`
	Time    string                       `json:"time"`
	Players []SyncLogReplayPlayerCompare `json:"players,omitempty"`
}

type SyncLogReplayPlayerCompare struct {
	PlayerID          int                     `json:"player_id"`
	LogResourceSum    float64                 `json:"log_resource_stockpile_word_1_candidate,omitempty"`
	ReplayWord1       uint32                  `json:"replay_word_1"`
	ResourceSumDiff   float64                 `json:"resource_stockpile_diff,omitempty"`
	LogObjectCount    int                     `json:"log_object_count_word_6_candidate"`
	ReplayWord6       uint32                  `json:"replay_word_6"`
	ObjectCountDiff   int64                   `json:"object_count_diff"`
	LogUnitTypeSum    int64                   `json:"log_unit_type_sum_word_2_candidate"`
	ReplayWord2       uint32                  `json:"replay_word_2"`
	UnitTypeSumDiff   int64                   `json:"unit_type_sum_diff"`
	LogStateSum       int64                   `json:"log_state_sum_word_3_candidate"`
	ReplayWord3       uint32                  `json:"replay_word_3"`
	StateSumDiff      int64                   `json:"state_sum_diff"`
	ReplayWord4       uint32                  `json:"replay_word_4"`
	Word4PerObject    float64                 `json:"word_4_per_log_object"`
	Word4Candidates   []SyncLogWord4Candidate `json:"word_4_candidates,omitempty"`
	LogPositionXY100  int64                   `json:"log_position_xy100_word_7_candidate"`
	ReplayWord7       uint32                  `json:"replay_word_7"`
	PositionXY100Diff int64                   `json:"position_xy100_diff"`
	LogObjectIDSum    int64                   `json:"log_object_id_sum_word_10_candidate"`
	ReplayWord10      uint32                  `json:"replay_word_10"`
	ObjectIDSumDiff   int64                   `json:"object_id_sum_diff"`
	Checksum          SyncLogChecksumEstimate `json:"log_checksum_estimate"`
}

type SyncLogReport struct {
	Path         string                `json:"path,omitempty"`
	Method       string                `json:"method"`
	Verification string                `json:"verification"`
	Summary      SyncLogSummary        `json:"summary"`
	Turns        []SyncLogTurn         `json:"turns,omitempty"`
	ObjectCensus []SyncLogObjectCensus `json:"object_census,omitempty"`
	StateCensus  []SyncLogStateCensus  `json:"state_census,omitempty"`
	CorpseCarry  []SyncLogCorpseCarry  `json:"corpse_carry,omitempty"`
	CorpseDecay  []SyncLogCorpseDecay  `json:"corpse_decay_baselines,omitempty"`
	RNGStreams   []SyncLogRNGStream    `json:"rng_streams,omitempty"`
	Comparison   *SyncLogReplayCompare `json:"replay_comparison,omitempty"`
	Warnings     []string              `json:"warnings,omitempty"`
}

func BuildSyncLogReport(path string, opts SyncLogOptions) (*SyncLogReport, error)

type SyncLogStateCensus struct {
	State      int      `json:"state"`
	Rows       int      `json:"rows"`
	HPZeroRows int      `json:"hp_zero_rows,omitempty"`
	CarrySum   int64    `json:"carry_sum"`
	Names      []string `json:"sample_names,omitempty"`
}

type SyncLogSummary struct {
	Lines             int    `json:"lines"`
	Turns             int    `json:"turns"`
	FirstTurn         int    `json:"first_turn,omitempty"`
	LastTurn          int    `json:"last_turn,omitempty"`
	FirstWorldTimeMS  int    `json:"first_world_time_ms,omitempty"`
	LastWorldTimeMS   int    `json:"last_world_time_ms,omitempty"`
	DurationMS        int    `json:"duration_ms,omitempty"`
	Duration          string `json:"duration,omitempty"`
	PlayerBlocks      int    `json:"player_blocks"`
	AttributeBlocks   int    `json:"attribute_blocks"`
	AttributeValues   int    `json:"attribute_values"`
	AttributeIndices  []int  `json:"attribute_indices,omitempty"`
	ObjectRows        int    `json:"object_rows"`
	DeclaredObjects   int    `json:"declared_objects"`
	RNGEvents         int    `json:"rng_events"`
	RNGSetSeeds       int    `json:"rng_set_seeds"`
	RNGRands          int    `json:"rng_rands"`
	UpdatePlayerLines int    `json:"update_player_lines"`
	PturnLines        int    `json:"pturn_lines"`
}

type SyncLogTurn struct {
	Turn          int                     `json:"turn"`
	WorldTimeMS   int                     `json:"world_time_ms"`
	WorldTime     string                  `json:"world_time"`
	Pturns        []int                   `json:"pturns,omitempty"`
	UpdatePlayers []int                   `json:"update_players,omitempty"`
	Players       []SyncLogPlayerSnapshot `json:"players,omitempty"`
	RNGStreams    []SyncLogRNGStream      `json:"rng_streams,omitempty"`
	ObjectCensus  []SyncLogObjectCensus   `json:"object_census,omitempty"`
}

type SyncLogWord4Candidate struct {
	Name        string `json:"name"`
	LogValue    int64  `json:"log_value"`
	ReplayWord4 uint32 `json:"replay_word_4"`
	Diff        int64  `json:"diff"`
}

type SyncOptions struct {
	// Limit caps emitted event rows (0 = all). Summary always covers every sync.
	Limit int
	// ChecksumsOnly emits only checksum-carrying syncs as rows.
	ChecksumsOnly bool
	// RawWords emits one row per DE checksum sample and player slot with all 11 words.
	RawWords bool
}

type SyncRawWordRow struct {
	SampleIndex  int      `json:"sample_index"`
	TimeMS       int      `json:"time_ms"`
	Time         string   `json:"time"`
	SourceOffset int      `json:"source_offset"`
	PlayerID     int      `json:"player_id"`
	Words        []uint32 `json:"words_u32"`
}

type SyncReport struct {
	Path          string            `json:"path,omitempty"`
	Method        string            `json:"method"`
	Verification  string            `json:"verification"`
	Summary       SyncSummary       `json:"summary"`
	DeltaShapes   []DeltaShape      `json:"delta_shapes,omitempty"`
	WordSemantics map[string]string `json:"checksum_word_semantics,omitempty"`
	Events        []SyncEvent       `json:"events,omitempty"`
	RawWords      []SyncRawWordRow  `json:"raw_words,omitempty"`
	StateDeltas   []SyncStateDelta  `json:"state_deltas,omitempty"`
	Warnings      []string          `json:"warnings,omitempty"`
}

func BuildSyncStream(path string, opts SyncOptions) (*SyncReport, error)

type SyncStateDelta struct {
	PlayerID            int    `json:"player_id"`
	PlayerLabel         string `json:"player_label"`
	FromTimeMS          int    `json:"from_time_ms"`
	FromTime            string `json:"from_time"`
	ToTimeMS            int    `json:"to_time_ms"`
	ToTime              string `json:"to_time"`
	ObjectCountDelta    int64  `json:"object_count_delta,omitempty"`
	UnitTypeSumDelta    int64  `json:"unit_type_sum_delta,omitempty"`
	ObjectIDSumDelta    int64  `json:"object_id_sum_delta,omitempty"`
	PositionSumDelta    int64  `json:"position_sum_delta,omitempty"`
	Word3Delta          int64  `json:"word_3_delta,omitempty"`
	Word4Delta          int64  `json:"word_4_delta,omitempty"`
	ScoreDelta          int64  `json:"word_9_score_candidate_delta,omitempty"`
	Kind                string `json:"kind"`
	UnitID              int    `json:"unit_id,omitempty"`
	UnitName            string `json:"unit_name,omitempty"`
	ReplacementUnitID   int    `json:"replacement_unit_id,omitempty"`
	ReplacementUnitName string `json:"replacement_unit_name,omitempty"`
	ObjectID            int64  `json:"object_id,omitempty"`
	Confidence          string `json:"confidence"`
}

type SyncSummary struct {
	SyncCount      int     `json:"sync_count"`
	DeltaOnly      int     `json:"delta_only_syncs"`
	ChecksumDE     int     `json:"checksum_de_syncs"`
	ChecksumLegacy int     `json:"checksum_legacy_syncs"`
	EmittedEvents  int     `json:"emitted_events"`
	FirstTimeMS    int     `json:"first_time_ms"`
	FirstTime      string  `json:"first_time"`
	LastTimeMS     int     `json:"last_time_ms"`
	LastTime       string  `json:"last_time"`
	DurationMS     int     `json:"duration_ms"`
	Duration       string  `json:"duration"`
	MinDeltaMS     int     `json:"min_delta_ms"`
	MaxDeltaMS     int     `json:"max_delta_ms"`
	MeanDeltaMS    float64 `json:"mean_delta_ms"`
}

type TauntCount struct {
	Number int    `json:"number"`
	Count  int    `json:"count"`
	Text   string `json:"text,omitempty"`
}

type TelemetryEvent struct {
	Prefix string            `json:"prefix"`
	Name   string            `json:"name,omitempty"`
	Fields map[string]string `json:"fields,omitempty"`
	Raw    string            `json:"raw"`
}

func ParseTelemetry(text string, prefixes []string) *TelemetryEvent

type TelemetryEventSchema struct {
	Required []string `json:"required,omitempty"`
	Optional []string `json:"optional,omitempty"`
}

type TelemetryExpectation struct {
	Name     string            `json:"name"`
	Fields   map[string]string `json:"fields,omitempty"`
	MinCount int               `json:"min_count,omitempty"`
}

type TelemetryIssue struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

type TelemetryReport struct {
	Path     string           `json:"path"`
	OK       bool             `json:"ok"`
	Events   []TelemetrySeen  `json:"events,omitempty"`
	Issues   []TelemetryIssue `json:"issues,omitempty"`
	Warnings []string         `json:"warnings,omitempty"`
}

func ValidateTelemetry(path string, contextPath string, schemaPath string) (*TelemetryReport, error)

func ValidateTelemetryFolder(root string, contextPath string, schemaPath string) ([]TelemetryReport, error)

type TelemetrySchema struct {
	Prefixes []string                        `json:"prefixes,omitempty"`
	Events   map[string]TelemetryEventSchema `json:"events"`
}

func LoadTelemetrySchema(path string) (*TelemetrySchema, error)

type TelemetrySeen struct {
	Time     string            `json:"time,omitempty"`
	PlayerID int               `json:"player_id,omitempty"`
	Prefix   string            `json:"prefix"`
	Name     string            `json:"name,omitempty"`
	Fields   map[string]string `json:"fields,omitempty"`
	Raw      string            `json:"raw"`
}

type UnitDeltaCount struct {
	UnitID   int    `json:"unit_id"`
	UnitName string `json:"unit_name,omitempty"`
	Count    int    `json:"count"`
}

type UnknownActionInfo struct {
	ActionID      int          `json:"action_id"`
	Count         int          `json:"count"`
	Players       []int        `json:"players_seen,omitempty"`
	FirstTimeMS   int          `json:"first_time_ms"`
	FirstTime     string       `json:"first_time"`
	LastTimeMS    int          `json:"last_time_ms"`
	LastTime      string       `json:"last_time"`
	PayloadShapes []DeltaShape `json:"payload_length_shapes,omitempty"`
	SampleHex     []string     `json:"sample_hex,omitempty"`
}

type UnknownActionSummary struct {
	ActionID     int      `json:"action_id"`
	Count        int      `json:"count"`
	PayloadBytes []int    `json:"payload_bytes,omitempty"`
	SampleHex    string   `json:"sample_hex,omitempty"`
	ReplayPaths  []string `json:"replay_paths,omitempty"`
}

type UnknownActionsReport struct {
	Path         string              `json:"path,omitempty"`
	Method       string              `json:"method"`
	Verification string              `json:"verification"`
	TotalActions int                 `json:"total_actions"`
	UnknownTotal int                 `json:"unknown_actions_total"`
	Groups       []UnknownActionInfo `json:"groups,omitempty"`
	Warnings     []string            `json:"warnings,omitempty"`
}

func BuildUnknownActions(path string, samples int) (*UnknownActionsReport, error)

type UnknownOperationSummary struct {
	OperationID int      `json:"operation_id"`
	Count       int      `json:"count"`
	ReplayPaths []string `json:"replay_paths,omitempty"`
}

type UnknownsReport struct {
	Root       string                    `json:"root"`
	Replays    int                       `json:"replays"`
	ActionIDs  []UnknownActionSummary    `json:"action_ids,omitempty"`
	Operations []UnknownOperationSummary `json:"operations,omitempty"`
	Warnings   []string                  `json:"warnings,omitempty"`
}

func BuildUnknowns(root string) (*UnknownsReport, error)

type ValueHit struct {
	QueryLabel string `json:"query_label"`
	QueryKind  string `json:"query_kind"`
	Space      string `json:"space"`
	Offset     int    `json:"offset"`
	End        int    `json:"end"`
	Bytes      int    `json:"bytes"`
	Region     string `json:"region,omitempty"`
	Status     string `json:"status,omitempty"`
	ContextHex string `json:"context_hex,omitempty"`
}

type ValueQuery struct {
	Label   string `json:"label"`
	Kind    string `json:"kind"`
	Pattern string `json:"pattern"`
	Bytes   int    `json:"bytes"`
}

type ValueScanOptions struct {
	Values        []string
	Hex           []string
	Text          []string
	NumericWidths []int
	Space         string
	Limit         int
}

type ValueScanReport struct {
	Path         string                 `json:"path,omitempty"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Summary      ValueScanSummary       `json:"summary"`
	Queries      []ValueQuery           `json:"queries"`
	Hits         []ValueHit             `json:"hits"`
	Warnings     []string               `json:"warnings,omitempty"`
}

func BuildValueScan(path string, opts ValueScanOptions) (*ValueScanReport, error)

type ValueScanSummary struct {
	InflatedHeaderBytes int `json:"inflated_header_bytes"`
	BodyBytes           int `json:"body_bytes"`
	QueryCount          int `json:"query_count"`
	HitCount            int `json:"hit_count"`
	Shown               int `json:"shown"`
}

type VerifyRunClaim struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Tier        string `json:"tier"`
	Source      string `json:"source"`
	Expected    string `json:"expected,omitempty"`
	Observed    string `json:"observed,omitempty"`
	Message     string `json:"message,omitempty"`
	Gotcha      string `json:"gotcha,omitempty"`
	Remediation string `json:"remediation,omitempty"`
}

type VerifyRunContract struct {
	Name               string                 `json:"name,omitempty"`
	ContextPath        string                 `json:"context,omitempty"`
	Scenario           *ScenarioExpectation   `json:"scenario,omitempty"`
	DataSet            *DataSetExpectation    `json:"data_set,omitempty"`
	Players            *PlayersExpectation    `json:"players,omitempty"`
	PlayerProfile      *ProfileExpectation    `json:"player_profile,omitempty"`
	Telemetry          []TelemetryExpectation `json:"telemetry,omitempty"`
	ForbiddenTelemetry []TelemetryExpectation `json:"forbidden_telemetry,omitempty"`
	Chat               []ChatExpectation      `json:"chat,omitempty"`
	ForbiddenChat      []ChatExpectation      `json:"forbidden_chat,omitempty"`
	Result             *ResultExpectation     `json:"result,omitempty"`
	RenderExpectations []RenderExpectation    `json:"render_expectations,omitempty"`
}

func LoadVerifyRunContract(path string) (*VerifyRunContract, error)

type VerifyRunReport struct {
	Replay   string               `json:"replay"`
	Contract string               `json:"contract,omitempty"`
	Name     string               `json:"name,omitempty"`
	OK       bool                 `json:"ok"`
	Summary  VerifyRunSummary     `json:"summary"`
	Story    *StoryReport         `json:"story,omitempty"`
	Profile  *PlayerProfileReport `json:"player_profile,omitempty"`
	Claims   []VerifyRunClaim     `json:"claims"`
	Warnings []string             `json:"warnings,omitempty"`
}

func VerifyRun(replayPath string, contractPath string) (*VerifyRunReport, error)

type VerifyRunSummary struct {
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Unknown int `json:"unknown"`
}

type XSTelemetryOptions struct {
	FoodBase      int
	Unused220Base int
}
    XSTelemetryOptions selects the authored carrier bases used by a probe.
    The defaults preserve the historical v3 probe contract; newer generators
    should pass the base they declare in their manifest.

type XSTelemetryReport struct {
	Path         string              `json:"path,omitempty"`
	Method       string              `json:"method"`
	Verification string              `json:"verification"`
	Summary      XSTelemetrySummary  `json:"summary"`
	Samples      []XSTelemetrySample `json:"samples,omitempty"`
	ChatMarkers  []ReplayEvent       `json:"chat_markers,omitempty"`
	Warnings     []string            `json:"warnings,omitempty"`
}

func BuildXSTelemetryProbe(path string) (*XSTelemetryReport, error)

func BuildXSTelemetryProbeWithOptions(path string, opts XSTelemetryOptions) (*XSTelemetryReport, error)

type XSTelemetrySample struct {
	TimeMS                int    `json:"time_ms"`
	Time                  string `json:"time"`
	PlayerID              int    `json:"player_id"`
	PlayerLabel           string `json:"player_label"`
	Word1                 uint32 `json:"word_1"`
	Word9                 uint32 `json:"word_9_score_candidate,omitempty"`
	Carrier               string `json:"carrier"`
	Attr20Value           int    `json:"attr_20_value"`
	Attr154Value          int    `json:"attr_154_value"`
	Attr43Value           int    `json:"attr_43_value"`
	KillEventsFromAttr154 int    `json:"kill_events_from_attr_154"`
	Confidence            string `json:"confidence"`
}

type XSTelemetrySummary struct {
	ChecksumSamples         int  `json:"checksum_samples"`
	DecodedSamples          int  `json:"decoded_samples"`
	FoodCarrierSamples      int  `json:"food_carrier_samples"`
	Unused220CarrierSamples int  `json:"unused_220_carrier_samples"`
	ChatMarkers             int  `json:"chat_markers"`
	FoodCarrierMoved        bool `json:"food_carrier_moved"`
	Unused220CarrierMoved   bool `json:"unused_220_carrier_moved"`
}
```

## aoe2kit/pkg/roadmap

```go
package roadmap // import "aoe2kit/pkg/roadmap"


CONSTANTS

const Version = "2026-09-27"

FUNCTIONS

func Domains() []string

TYPES

type CRUDCapability struct {
	Status string `json:"status"`
	Mode   string `json:"mode,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type CRUDReport struct {
	Version      string                 `json:"version"`
	Domain       string                 `json:"domain"`
	Method       string                 `json:"method"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Summary      CRUDSummary            `json:"summary"`
	Rows         []CRUDRow              `json:"rows"`
	Notes        []string               `json:"notes,omitempty"`
}

func CRUDMatrix(domain string) (CRUDReport, error)

type CRUDRow struct {
	Domain       string         `json:"domain"`
	Section      string         `json:"section"`
	Create       CRUDCapability `json:"create"`
	Read         CRUDCapability `json:"read"`
	Update       CRUDCapability `json:"update"`
	Delete       CRUDCapability `json:"delete"`
	Verification string         `json:"verification"`
	Evidence     string         `json:"evidence"`
	Next         string         `json:"next"`
}

type CRUDSummary struct {
	Domains             []string `json:"domains"`
	Rows                int      `json:"rows"`
	CreateFull          int      `json:"create_full"`
	ReadFull            int      `json:"read_full"`
	UpdateFull          int      `json:"update_full"`
	DeleteFull          int      `json:"delete_full"`
	PartialCapabilities int      `json:"partial_capabilities"`
	Unsupported         int      `json:"unsupported"`
	NotApplicable       int      `json:"not_applicable"`
	NeedsEngineRun      int      `json:"needs_engine_run"`
}

type DarkFrontier struct {
	Domain       string `json:"domain"`
	Region       string `json:"region"`
	Priority     int    `json:"priority"`
	Known        string `json:"known"`
	Unknown      string `json:"unknown"`
	Tools        string `json:"tools"`
	Next         string `json:"next"`
	Verification string `json:"verification"`
}

type DarkReport struct {
	Version      string                 `json:"version"`
	Domain       string                 `json:"domain"`
	Method       string                 `json:"method"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Summary      DarkSummary            `json:"summary"`
	Frontiers    []DarkFrontier         `json:"frontiers"`
	Notes        []string               `json:"notes,omitempty"`
}

func DarkBytes(domain string) (DarkReport, error)

type DarkSummary struct {
	Domains   []string `json:"domains"`
	Frontiers int      `json:"frontiers"`
	Priority1 int      `json:"priority_1"`
	Priority2 int      `json:"priority_2"`
	Priority3 int      `json:"priority_3"`
}

type MatrixReport struct {
	Version      string                 `json:"version"`
	Domain       string                 `json:"domain"`
	Method       string                 `json:"method"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Summary      MatrixSummary          `json:"summary"`
	Rows         []RWDRow               `json:"rows"`
	Notes        []string               `json:"notes,omitempty"`
}

func RWDMatrix(domain string) (MatrixReport, error)

type MatrixSummary struct {
	Domains       []string `json:"domains"`
	Rows          int      `json:"rows"`
	ReadFull      int      `json:"read_full"`
	WriteFull     int      `json:"write_full"`
	CreateFull    int      `json:"create_full"`
	DeleteFull    int      `json:"delete_full"`
	PartialRows   int      `json:"partial_rows"`
	Unsupported   int      `json:"unsupported"`
	NotApplicable int      `json:"not_applicable"`
}

type RWDRow struct {
	Domain       string `json:"domain"`
	Section      string `json:"section"`
	Read         string `json:"read"`
	Write        string `json:"write"`
	Create       string `json:"create"`
	Delete       string `json:"delete"`
	Verification string `json:"verification"`
	Evidence     string `json:"evidence"`
	Next         string `json:"next"`
}
```

## aoe2kit/pkg/rpgshop

```go
package rpgshop // import "aoe2kit/pkg/rpgshop"


CONSTANTS

const (
	DemoScenarioName = "A2K RPG Shop Demo"
)

FUNCTIONS

func DemoRecipe(xsPath string, timestamp int) scenario.Recipe
func XSModule() string

TYPES

type DemoEarning struct {
	Signal           string `json:"signal"`
	Players          []int  `json:"players"`
	EnemyPlayers     []int  `json:"enemy_players"`
	GoldPerKill      int    `json:"gold_per_kill"`
	XPPerKill        int    `json:"xp_per_kill"`
	LevelCurve       string `json:"level_curve"`
	HeroAttribution  string `json:"hero_attribution"`
	PollTriggerName  string `json:"poll_trigger_name"`
	PollIntervalNote string `json:"poll_interval_note"`
}

type DemoOptions struct {
	OutputDir    string
	ScenarioName string
	Timestamp    int
}

type DemoReport struct {
	OutputDir          string               `json:"output_dir"`
	ScenarioPath       string               `json:"scenario_path"`
	BaseScenarioPath   string               `json:"base_scenario_path"`
	RecipePath         string               `json:"recipe_path"`
	XSPath             string               `json:"xs_path"`
	InstructionsPath   string               `json:"instructions_path"`
	ManifestPath       string               `json:"manifest_path"`
	ScenarioSHA256     string               `json:"scenario_sha256"`
	XSSHA256           string               `json:"xs_sha256"`
	BlankReport        scenario.BlankReport `json:"blank_report"`
	PatchReport        scenario.PatchReport `json:"patch_report"`
	Variables          []DemoVariable       `json:"variables"`
	ShopItems          []DemoShopItem       `json:"shop_items"`
	Earning            DemoEarning          `json:"earning"`
	VerificationClaims []string             `json:"verification_claims"`
	KnownGaps          []string             `json:"known_gaps"`
	SemanticLint       scenario.LintReport  `json:"semantic_lint"`
	Files              map[string]string    `json:"files"`
}

func BuildHeroShopDemo(opts DemoOptions) (*DemoReport, error)

type DemoShopItem struct {
	Scope      string `json:"scope"`
	Item       string `json:"item"`
	Player     int    `json:"player"`
	Hero       int    `json:"hero"`
	TokenUnit  int    `json:"token_unit"`
	HeroUnit   int    `json:"hero_unit"`
	Cost       int    `json:"cost"`
	UnlockNote string `json:"unlock_note"`
}

type DemoVariable struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Scope string `json:"scope"`
}
```

## aoe2kit/pkg/scenario

```go
package scenario // import "aoe2kit/pkg/scenario"


CONSTANTS

const (
	ShopCatalogDefaultPlayer    = 1
	ShopCatalogDefaultTrainTime = 1
	ShopCatalogDefaultHotkey    = 0
	ShopCatalogDefaultAreaMin   = 0
	ShopCatalogDefaultAreaMax   = 479
)
const (
	MinScenarioMapSize = 80
	MaxScenarioMapSize = 480
)

VARIABLES

var DE158Layout = Layout{
	Name:      "aoe2de-scenario",
	Revisions: []LayoutRevision{{Name: "file-map-v1", Versions: []string{"1.58", "1.59"}, Status: "corpus_verified"}},
	Sections: []LayoutSection{
		{Name: "FileHeader", Fields: []LayoutField{
			{Name: "version", Encoding: EncodingCString4, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved headers"}},
			{Name: "header_length", Encoding: EncodingU32, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved headers"}},
			{Name: "savable", Encoding: EncodingS32, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved headers"}},
			{Name: "timestamp_of_last_save", Encoding: EncodingU32, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved headers"}},
			{Name: "scenario_instructions", Encoding: EncodingString32, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved headers"}},
			{Name: "player_count", Encoding: EncodingU32, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved headers"}},
			{Name: "unknown_value", Encoding: EncodingU32, Opaque: true, Evidence: Evidence{Tier: "dark", Source: "corpus value only"}},
			{Name: "unknown_value_2", Encoding: EncodingU32, Opaque: true, Evidence: Evidence{Tier: "dark", Source: "corpus value only"}},
			{Name: "amount_of_unknown_numbers", Encoding: EncodingU32, Opaque: true, Evidence: Evidence{Tier: "corpus_verified", Source: "count precedes list"}},
			{Name: "unknown_numbers", Encoding: EncodingU32, Count: -1, Opaque: true, Evidence: Evidence{Tier: "corpus_verified", Source: "counted header list"}},
			{Name: "creator_name", Encoding: EncodingString32, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved headers"}},
			{Name: "trigger_count", Encoding: EncodingU32, Evidence: Evidence{Tier: "corpus_verified", Source: "header/body agreement"}},
		}},
		{Name: "Map", Fields: []LayoutField{
			{Name: "string_starter_1", Encoding: EncodingBytes, Count: 2, Opaque: true, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved map sections"}},
			{Name: "water_definition", Encoding: EncodingString16, Evidence: Evidence{Tier: "corpus_verified", Source: "editor calibration round3"}},
			{Name: "string_starter_2", Encoding: EncodingBytes, Count: 2, Opaque: true, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved map sections"}},
			{Name: "map_color_mood", Encoding: EncodingString16, Evidence: Evidence{Tier: "corpus_verified", Source: "editor calibration round3"}},
			{Name: "string_starter_3", Encoding: EncodingBytes, Count: 2, Opaque: true, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved map sections"}},
			{Name: "script_name", Encoding: EncodingString16, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved map sections"}},
			{Name: "collide_and_correct", Encoding: EncodingU8, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved map sections"}},
			{Name: "villager_force_drop", Encoding: EncodingU8, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved map sections"}},
			{Name: "initial_player_views", Encoding: EncodingS32, Count: 32, Evidence: Evidence{Tier: "corpus_verified", Source: "fixed 16 x/y pairs"}},
			{Name: "lock_coop_alliances", Encoding: EncodingU8, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved map sections"}},
			{Name: "per_player_population_cap", Encoding: EncodingU32, Count: 16, Evidence: Evidence{Tier: "corpus_verified", Source: "fixed player array"}},
			{Name: "secondary_game_modes", Encoding: EncodingU32, Evidence: Evidence{Tier: "engine_verified", Source: "editor calibration round3"}},
			{Name: "unknown_5", Encoding: EncodingBytes, Count: 4, Opaque: true, Evidence: Evidence{Tier: "dark", Source: "corpus bytes only"}},
			{Name: "unknown_4", Encoding: EncodingBytes, Count: 4, Opaque: true, Evidence: Evidence{Tier: "dark", Source: "corpus bytes only"}},
			{Name: "no_waves_on_shore", Encoding: EncodingS8, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved map sections"}},
			{Name: "map_width", Encoding: EncodingS32, Evidence: Evidence{Tier: "corpus_verified", Source: "terrain count agreement"}},
			{Name: "map_height", Encoding: EncodingS32, Evidence: Evidence{Tier: "corpus_verified", Source: "terrain count agreement"}},
			{Name: "terrain_data", Encoding: EncodingBytes, Count: -1, Evidence: Evidence{Tier: "corpus_verified", Source: "width x height x 5-byte cells"}},
		}},
	},
	MirrorGroups: []MirrorGroup{
		{Name: "player_diplomacy", Authoritative: "Diplomacy.matrix", Derived: []string{"Players[].diplomacy", "Units[].diplomacy_for_interaction"}, Policy: "authoritative_matrix_projects_to_derived_views"},
		{Name: "allied_victory", Authoritative: "Diplomacy.allied_victory", Derived: []string{"Players[].allied_victory", "Units[].aok_allied_victory"}, Policy: "authoritative_array_projects_to_derived_views"},
	},
}
    DE158Layout is the first independently owned layout slice. 1.59 is listed
    only because the corpus currently proves these sections are shared; it is
    not an assumption that every later section is unchanged.


FUNCTIONS

func BlankScenarioSeedBytes() ([]byte, error)
func ConditionTypeForName(name string) (int, bool)
func ConditionTypeName(t int) string
    ConditionTypeName maps trigger condition type ids to names (DE editor
    order).

func DecodeDataHeaderNode(data []byte) (DataHeaderLayout, *parsedNode, int, error)
func DecodeDiplomacyNode(data []byte) (DiplomacyLayout, *parsedNode, int, error)
func DecodeGlobalVictoryNode(data []byte) (GlobalVictoryLayout, *parsedNode, int, error)
func DecodeMapLayoutNode(data []byte) (MapLayout, *parsedNode, int, error)
    DecodeMapLayoutNode is the mutable projection used while the writer is
    being migrated. Map bytes are consumed by DecodeMapLayout's typed codec;
    this projection preserves field boundaries for existing mutation recipes.

func DecodeScenarioHeaderNode(data []byte) (ScenarioHeader, *parsedNode, int, error)
    DecodeScenarioHeaderNode exposes the typed header through the compatibility
    mutation surface used by the writer. The bytes are decoded by the owned
    header codec above; this node is only a mutable projection, not a second
    layout interpreter.

func DeflateRaw(payload []byte) ([]byte, error)
func EffectTypeForOp(op string) (int, bool)
func EffectTypeName(effectType int) string
func InflateRaw(compressed []byte) ([]byte, error)
func InflatedBodyFile(path string) ([]byte, error)
func ParseInflatedTriggers(version string, data []byte) (*triggergraph.Graph, int, error)
    ParseInflatedTriggers parses an extracted DE scenario Triggers section.
    Replays can embed this section without the scenario file header; keeping
    this entry point separate lets replay readers reuse the versioned scenario
    schema without pretending the surrounding replay header is a scenario file.

func ReadScenarioVersion(data []byte) (string, error)
    ReadScenarioVersion reads only the FileHeader. It is intentionally cheap so
    corpus commands can classify unsupported versions before attempting a body
    parse.

func ReadScenarioVersionFile(path string) (string, error)
func ScenarioMapPresetSizesString() string
func SortedShopCatalogCostResources(costs []ShopCatalogItemCost) []int
func StringContainsDisconnectRecipeFile(path string, text string) ([]int, ReferenceReport, Recipe, error)
func StringDisconnectRecipeFile(path string, stringID int) (ReferenceReport, Recipe, error)
func StringPrefixDisconnectRecipeFile(path string, prefix string) ([]int, ReferenceReport, Recipe, error)
func StringTextDisconnectRecipeFile(path string, text string) (int, ReferenceReport, Recipe, error)
func SupportsReadVersion(version string) bool
func SupportsWriteVersion(version string) bool
func TriggerContainsDisconnectRecipeFile(path string, text string) ([]int, ReferenceReport, Recipe, error)
func TriggerDisconnectRecipeFile(path string, targetID int) (ReferenceReport, Recipe, error)
func TriggerNameDisconnectRecipeFile(path string, name string) (int, ReferenceReport, Recipe, error)
func TriggerPrefixDisconnectRecipeFile(path string, prefix string) ([]int, ReferenceReport, Recipe, error)
func UnitCaptionContainsDisconnectRecipeFile(path string, text string, targetPlayer *int) ([]int, ReferenceReport, Recipe, error)
func UnitCaptionDisconnectRecipeFile(path string, caption string, targetPlayer *int) (int, ReferenceReport, Recipe, error)
func UnitCaptionPrefixDisconnectRecipeFile(path string, prefix string, targetPlayer *int) ([]int, ReferenceReport, Recipe, error)
func UnitDisconnectRecipeFile(path string, referenceID int) (ReferenceReport, Recipe, error)
func UnitTypeDisconnectRecipeFile(path string, unitConst int, targetPlayer *int) ([]int, ReferenceReport, Recipe, error)
func UnitsPlayerDisconnectRecipeFile(path string, player int) ([]int, ReferenceReport, Recipe, error)
func ValidateScenarioMapSize(width, height int) error
func ValidateSemanticMirrors(doc SemanticDocument) error
    ValidateSemanticMirrors enforces the first migrated mirror groups.
    The authoritative copy is the diplomacy matrix/allied-victory arrays;
    player projections and unit diplomacy mirrors are derived views.

func VariableContainsDisconnectRecipeFile(path string, text string) ([]int, ReferenceReport, Recipe, error)
func VariableDisconnectRecipeFile(path string, variableID int) (ReferenceReport, Recipe, error)
func VariableNameDisconnectRecipeFile(path string, name string) (int, ReferenceReport, Recipe, error)
func VariablePrefixDisconnectRecipeFile(path string, prefix string) ([]int, ReferenceReport, Recipe, error)
func XSCarrierContent(message string) string
func DecodeBackgroundImageNode(data []byte) (*parsedNode, int, error)
    DecodeBackgroundImageNode owns the image frame. Empty editor images are a
    16-byte frame; populated images carry one bitmap-info record followed by
    variable color and pixel payloads.

func DecodeCinematicsNode(data []byte) (*parsedNode, int, error)
func DecodeConditionNode(data []byte) (*parsedNode, int, error)
func DecodeEffectNode(data []byte) (*parsedNode, int, error)
func DecodeFilesNode(data []byte) (*parsedNode, int, error)
    DecodeFilesNode owns the script/AI carrier while accepting the editor's
    valid path-only form. Optional tails are parsed only when bytes remain.

func DecodeMessagesNode(data []byte) (*parsedNode, int, error)
func DecodeOptionsNode(data []byte) (*parsedNode, int, error)
func DecodePlayerDataFourNode(data []byte) (*parsedNode, int, error)
func DecodePlayerDataThreeNode(data []byte) (*parsedNode, int, error)
func DecodePlayerDataTwoNode(data []byte) (*parsedNode, int, error)
    DecodePlayerDataTwoNode owns the fixed player metadata section. AI payloads
    and resource mirrors are decoded structurally, while the eight-byte AI
    prefix remains opaque because its semantics are not established.

func DecodePlayerUnitsNode(data []byte, has159Byte bool) (*parsedNode, int, error)
func DecodeTriggerNode(data []byte) (*parsedNode, int, error)
    DecodeTriggerNode is the owned 1.58/1.59 trigger record codec. Trigger
    records are framing, not a generic bag of fields: the effect and condition
    counts delimit typed variable-length arrays in the middle of the record.

func DecodeUnitStructNode(data []byte, has159Byte bool) (*parsedNode, int, error)
    DecodeUnitStructNode decodes the stable unit payload. DE 1.59 inserts one
    discriminator byte immediately before caption_string_id; the caller tells us
    which proven layout the enclosing version selected.

func DecodeVariableNode(data []byte) (*parsedNode, int, error)

TYPES

type AIFileInfo struct {
	Index         int    `json:"index"`
	Name          string `json:"name,omitempty"`
	ContentBytes  int    `json:"content_bytes,omitempty"`
	ContentSHA256 string `json:"content_sha256,omitempty"`
}

type AlliedVictoryRecipe struct {
	Player  int  `json:"player"`
	Enabled bool `json:"enabled"`
}

type AnalysisReport struct {
	Path              string              `json:"path,omitempty"`
	Version           string              `json:"version"`
	Verification      string              `json:"verification"`
	DataSet           ScenarioDataSet     `json:"data_set"`
	Scale             ScenarioScale       `json:"scale"`
	DisplayTechniques DisplayTechniques   `json:"display_techniques"`
	EconomySignals    []EconomySignal     `json:"economy_signals,omitempty"`
	Watermarks        []WatermarkSignal   `json:"watermarks,omitempty"`
	EffectTypes       []EffectTypeSummary `json:"effect_types,omitempty"`
}

type BeforeAfter[T any] struct {
	Before T `json:"before"`
	After  T `json:"after"`
}

type BlankOptions struct {
	PlayerCount        int
	HumanSlots         int
	MapWidth           int
	MapHeight          int
	Timestamp          int
	ClearTriggers      bool
	KeepSeedTriggers   bool
	DummyStarters      bool
	NoStarters         bool // Explicit editor-empty opt-out; playable blanks include starters.
	DummyUnit          int
	DummyStartX        float64
	DummyStartY        float64
	DummySpacing       float64
	StartingResources  bool
	StartingFood       int
	StartingWood       int
	StartingGold       int
	StartingStone      int
	StartingTradeGoods int
	InactiveRestHuman  bool
	GaiaActive         bool
	NoConquest         bool
	CloseoutSeconds    int
	CloseoutPlayer     int
}

type BlankReport struct {
	Output             string `json:"output"`
	SeedSHA256         string `json:"seed_sha256"`
	Version            string `json:"version"`
	PlayerCount        int    `json:"player_count"`
	HumanSlots         int    `json:"human_slots"`
	MapWidth           int    `json:"map_width"`
	MapHeight          int    `json:"map_height"`
	TileCount          int    `json:"tile_count"`
	ClearTriggers      bool   `json:"clear_triggers"`
	DummyStarters      bool   `json:"dummy_starters"`
	DummyUnit          int    `json:"dummy_unit,omitempty"`
	GaiaActive         bool   `json:"gaia_active"`
	NoConquest         bool   `json:"no_conquest"`
	CloseoutSeconds    int    `json:"closeout_seconds,omitempty"`
	CloseoutPlayer     int    `json:"closeout_player,omitempty"`
	TriggerCountBefore int    `json:"trigger_count_before"`
	TriggerCountAfter  int    `json:"trigger_count_after"`
	UnitCountBefore    int    `json:"unit_count_before"`
	UnitCountAfter     int    `json:"unit_count_after"`
	RebuildOK          bool   `json:"rebuild_ok"`
	InvariantOK        bool   `json:"invariant_ok"`
	EditorParityOK     bool   `json:"editor_parity_ok"`
	EditorParityNote   string `json:"editor_parity_note,omitempty"`
	Verification       string `json:"verification"`
}

func WriteBlankScenarioFile(output string, opts BlankOptions) (BlankReport, error)

type ByteChangeInfo struct {
	Offset      int    `json:"offset"`
	OffsetHex   string `json:"offset_hex"`
	BeforeUint8 uint8  `json:"before_uint8"`
	AfterUint8  uint8  `json:"after_uint8"`
	BeforeASCII string `json:"before_ascii"`
	AfterASCII  string `json:"after_ascii"`
}

type ByteCoverage struct {
	Space     string     `json:"space"`
	Name      string     `json:"name"`
	Start     int        `json:"start"`
	End       int        `json:"end"`
	Bytes     int        `json:"bytes"`
	Accounted int        `json:"accounted_bytes"`
	Named     int        `json:"named_bytes"`
	Dark      int        `json:"dark_bytes"`
	Gaps      int        `json:"gap_bytes"`
	Status    string     `json:"status"`
	DarkSpans []DarkSpan `json:"dark_spans,omitempty"`
}

type ByteDiffDecodes struct {
	Int16LE *BeforeAfter[int16]  `json:"int16_le,omitempty"`
	Int32LE *BeforeAfter[int32]  `json:"int32_le,omitempty"`
	Uint8   *BeforeAfter[uint8]  `json:"uint8,omitempty"`
	ASCII   *BeforeAfter[string] `json:"ascii,omitempty"`
	FieldLE *BeforeAfter[int32]  `json:"field_int32_le,omitempty"`
}

type ByteDiffOptions struct {
	IncludeNoise bool `json:"include_noise"`
}

type ByteDiffReport struct {
	BeforePath       string           `json:"before_path,omitempty"`
	AfterPath        string           `json:"after_path,omitempty"`
	BeforeBytes      int              `json:"before_inflated_bytes"`
	AfterBytes       int              `json:"after_inflated_bytes"`
	Mode             string           `json:"mode"`
	LengthDelta      int              `json:"length_delta"`
	FirstDivergence  int              `json:"first_divergence"`
	Warning          string           `json:"warning,omitempty"`
	NoiseSuppressed  []SuppressedRun  `json:"noise_suppressed,omitempty"`
	Changes          []ByteDiffRun    `json:"changes,omitempty"`
	Verification     string           `json:"verification"`
	Notes            []string         `json:"notes,omitempty"`
	ComparedByteSpan ComparedByteSpan `json:"compared_byte_span"`
}

func ByteDiffBodies(before, after []byte, opts ByteDiffOptions) ByteDiffReport

func ByteDiffFiles(beforePath, afterPath string, opts ByteDiffOptions) (ByteDiffReport, error)

type ByteDiffRun struct {
	Start       int              `json:"start"`
	StartHex    string           `json:"start_hex"`
	End         int              `json:"end"`
	Length      int              `json:"length"`
	BeforeHex   string           `json:"before_hex,omitempty"`
	AfterHex    string           `json:"after_hex,omitempty"`
	BeforeText  string           `json:"before_ascii,omitempty"`
	AfterText   string           `json:"after_ascii,omitempty"`
	Decodes     ByteDiffDecodes  `json:"decodes"`
	ByteChanges []ByteChangeInfo `json:"byte_changes,omitempty"`
}

type CaptionTechnique struct {
	Detected bool `json:"detected"`
	Count    int  `json:"count"`
}

type CinematicsSettings struct {
	Pregame string `json:"pregame,omitempty"`
	Victory string `json:"victory,omitempty"`
	Loss    string `json:"loss,omitempty"`
}

type ComparedByteSpan struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type ConditionRecipe struct {
	Op                             string `json:"op"`
	Timer                          *int   `json:"timer,omitempty"`
	UnitObject                     *int   `json:"unit_object,omitempty"`
	Quantity                       *int   `json:"quantity,omitempty"`
	Attribute                      *int   `json:"attribute,omitempty"`
	ObjectList                     *int   `json:"object_list,omitempty"`
	SourcePlayer                   *int   `json:"source_player,omitempty"`
	Technology                     *int   `json:"technology,omitempty"`
	AreaX1                         *int   `json:"area_x1,omitempty"`
	AreaY1                         *int   `json:"area_y1,omitempty"`
	AreaX2                         *int   `json:"area_x2,omitempty"`
	AreaY2                         *int   `json:"area_y2,omitempty"`
	ObjectGroup                    *int   `json:"object_group,omitempty"`
	ObjectType                     *int   `json:"object_type,omitempty"`
	ObjectState                    *int   `json:"object_state,omitempty"`
	Variable                       *int   `json:"variable,omitempty"`
	Comparison                     *int   `json:"comparison,omitempty"`
	TargetPlayer                   *int   `json:"target_player,omitempty"`
	UnitAIAction                   *int   `json:"unit_ai_action,omitempty"`
	IncludeChangeableWeaponObjects *int   `json:"include_changeable_weapon_objects,omitempty"`
	Inverted                       *int   `json:"inverted,omitempty"`
}

type ConditionSummary struct {
	TriggerIndex   int            `json:"trigger_index,omitempty"`
	TriggerName    string         `json:"trigger_name,omitempty"`
	ConditionIndex int            `json:"condition_index,omitempty"`
	Type           int            `json:"type"`
	TypeName       string         `json:"type_name"`
	KnownFields    map[string]any `json:"known_fields,omitempty"`
}

type CoverageCorpusError struct {
	Path    string `json:"path"`
	Version string `json:"version,omitempty"`
	Status  string `json:"status"`
	Error   string `json:"error"`
}

type CoverageCorpusOptions struct {
	Strict bool
}

type CoverageCorpusReport struct {
	Roots  []string              `json:"roots"`
	Files  int                   `json:"files"`
	Parsed int                   `json:"parsed"`
	Errors []CoverageCorpusError `json:"errors,omitempty"`
	Spans  []CoverageCorpusSpan  `json:"spans"`
}
    CoverageCorpusReport summarizes dark-byte spans without retaining parsed
    scenarios. CoverageFile is called one file at a time so large scenario
    corpora remain bounded by the largest individual scenario.

func CoverageCorpus(roots ...string) (CoverageCorpusReport, error)
    CoverageCorpus walks roots recursively and analyzes every .aoe2scenario.
    A malformed or unsupported file is reported and does not abort the corpus.

func CoverageCorpusWithOptions(opts CoverageCorpusOptions, roots ...string) (CoverageCorpusReport, error)

type CoverageCorpusSpan struct {
	Version           string   `json:"version"`
	Path              string   `json:"path"`
	Bytes             int      `json:"bytes"`
	Occurrences       int      `json:"occurrences"`
	Files             int      `json:"files"`
	SHA256            []string `json:"sha256"`
	NonZeroBytes      []int    `json:"nonzero_bytes"`
	MinTriggerCount   int      `json:"min_trigger_count"`
	MaxTriggerCount   int      `json:"max_trigger_count"`
	MinConditionCount int      `json:"min_condition_count"`
	MaxConditionCount int      `json:"max_condition_count"`
}

type CoverageReport struct {
	Path              string         `json:"path,omitempty"`
	Version           string         `json:"version"`
	Verification      string         `json:"verification"`
	FileBytes         int            `json:"file_bytes"`
	HeaderBytes       int            `json:"header_bytes"`
	InflatedBodyBytes int            `json:"inflated_body_bytes"`
	TriggerCount      int            `json:"trigger_count"`
	ConditionCount    int            `json:"condition_count"`
	AccountedBytes    int            `json:"accounted_bytes"`
	NamedBytes        int            `json:"named_bytes"`
	DarkBytes         int            `json:"dark_bytes"`
	GapBytes          int            `json:"gap_bytes"`
	Sections          []ByteCoverage `json:"sections"`
	Notes             []string       `json:"notes,omitempty"`
}
    CoverageReport accounts for parsed scenario bytes without treating
    successful structural parsing as semantic understanding.

func CoverageFile(path string) (CoverageReport, error)
    CoverageFile reports structural accounting plus anonymous/unknown spans.
    A complete parse can have zero gaps while retaining semantic dark bytes.

type CreateKillLoopSample struct {
	TriggerIndex  int    `json:"trigger_index"`
	TriggerName   string `json:"trigger_name,omitempty"`
	CreateObjects int    `json:"create_objects"`
	KillObjects   int    `json:"kill_objects"`
}

type CreateKillLoopTechnique struct {
	Detected bool                   `json:"detected"`
	Count    int                    `json:"count"`
	Samples  []CreateKillLoopSample `json:"samples,omitempty"`
}

type DarkSpan struct {
	Path         string `json:"path"`
	Start        int    `json:"start"`
	End          int    `json:"end"`
	Bytes        int    `json:"bytes"`
	SHA256       string `json:"sha256,omitempty"`
	NonZeroBytes int    `json:"nonzero_bytes,omitempty"`
	HexPrefix    string `json:"hex_prefix,omitempty"`
}
    DarkSpan identifies the parser field responsible for a conservative dark
    byte range. Repeated structures may produce the same field path at different
    offsets; offsets are the identity.

type DataHeaderLayout struct {
	NextUnitID       uint32
	Version          float32
	PlayerCapacity   int32
	GaiaPlayerIndex  int32
	TribeNames       []string
	PlayerNameIDs    []int32
	Players          []PlayerDataLayout
	LockCivilization []uint32
	LockPersonality  []uint32
	Opaque           []byte
	Filename         string
}
    DataHeaderLayout is the owned player/header slice. The player records are
    deliberately typed, while the two fixed opaque tails retain their bytes.

type DeletePlanReport struct {
	Path              string              `json:"path,omitempty"`
	Version           string              `json:"version"`
	Verification      string              `json:"verification"`
	Request           DeletePlanRequest   `json:"request"`
	CanDelete         bool                `json:"can_delete"`
	CanTombstone      bool                `json:"can_tombstone,omitempty"`
	Strategy          string              `json:"strategy"`
	ReferenceSummary  ReferenceSummary    `json:"reference_summary"`
	BlockingRefs      []ScenarioReference `json:"blocking_refs,omitempty"`
	ResolvedIDs       []int               `json:"resolved_ids,omitempty"`
	ResolvedSets      map[string][]int    `json:"resolved_sets,omitempty"`
	SuggestedRecipe   map[string]any      `json:"suggested_recipe,omitempty"`
	CleanupCommand    string              `json:"cleanup_command,omitempty"`
	CleanupRecipe     map[string]any      `json:"cleanup_recipe,omitempty"`
	StructuralCaveats []string            `json:"structural_caveats,omitempty"`
}

func DeletePlanFile(path string, request DeletePlanRequest) (DeletePlanReport, error)

type DeletePlanRequest struct {
	Kind          string   `json:"kind"`
	ID            int      `json:"id"`
	TriggerID     *int     `json:"trigger_id,omitempty"`
	ChildIndex    *int     `json:"child_index,omitempty"`
	TargetName    string   `json:"target_name,omitempty"`
	TargetPrefix  string   `json:"target_prefix,omitempty"`
	TargetCaption string   `json:"target_caption,omitempty"`
	TargetText    string   `json:"target_text,omitempty"`
	AreaX1        *float64 `json:"area_x1,omitempty"`
	AreaY1        *float64 `json:"area_y1,omitempty"`
	AreaX2        *float64 `json:"area_x2,omitempty"`
	AreaY2        *float64 `json:"area_y2,omitempty"`
	Player        *int     `json:"player,omitempty"`
	UnitConst     *int     `json:"unit_const,omitempty"`
}

type DeployCheckFinding struct {
	Severity     string `json:"severity"`
	What         string `json:"what"`
	Path         string `json:"path,omitempty"`
	Line         int    `json:"line,omitempty"`
	Fix          string `json:"fix"`
	FactID       string `json:"fact_id,omitempty"`
	FactTier     string `json:"fact_tier,omitempty"`
	VerifiedDate string `json:"verified_date,omitempty"`
	FixtureRef   string `json:"fixture_ref,omitempty"`
}

type DeployCheckOptions struct {
	IncludeProvisional bool
}

type DeployCheckReport struct {
	Path         string                   `json:"path,omitempty"`
	DeployTree   string                   `json:"deploy_tree"`
	OK           bool                     `json:"ok"`
	Verification string                   `json:"verification"`
	XS           XSDeployAttachment       `json:"xs"`
	Findings     []DeployCheckFinding     `json:"findings,omitempty"`
	Analysis     *xsauthor.AnalysisReport `json:"analysis,omitempty"`
}

func DeployCheckFile(path, deployTree string) (DeployCheckReport, error)

func DeployCheckFileWithOptions(path, deployTree string, opts DeployCheckOptions) (DeployCheckReport, error)

type DescribeOptions struct {
	Sections []string
	Full     bool
}

type Description struct {
	Path            string        `json:"path,omitempty"`
	Version         string        `json:"version"`
	PlayerCount     int           `json:"player_count"`
	HeaderBytes     int           `json:"header_bytes"`
	CompressedBytes int           `json:"compressed_body_bytes"`
	InflatedBytes   int           `json:"inflated_body_bytes"`
	SectionIndex    []SectionInfo `json:"section_index"`
	Players         []PlayerInfo  `json:"players,omitempty"`
	Triggers        *TriggerInfo  `json:"triggers,omitempty"`
	Units           *UnitInfo     `json:"units,omitempty"`
	Map             *MapInfo      `json:"map,omitempty"`
	AI              []AIFileInfo  `json:"ai,omitempty"`
	Sections        []SectionDump `json:"sections,omitempty"`
}

type DiffChange struct {
	Kind         string `json:"kind"`
	Field        string `json:"field"`
	Player       string `json:"player,omitempty"`
	SectionIndex *int   `json:"section_index,omitempty"`
	IndexBase    string `json:"index_base,omitempty"`
	Before       any    `json:"before,omitempty"`
	After        any    `json:"after,omitempty"`
	Detail       string `json:"detail,omitempty"`
}

type DiffOptions struct {
	AllFields bool
}

type DiffReport struct {
	Before  string               `json:"before"`
	After   string               `json:"after"`
	Same    bool                 `json:"same"`
	Changes []DiffChange         `json:"changes"`
	Fields  []StructureFieldDiff `json:"fields,omitempty"`
}

func Diff(before, after *File) DiffReport

func DiffFiles(beforePath, afterPath string) (DiffReport, error)

func DiffFilesWithOptions(beforePath, afterPath string, opts DiffOptions) (DiffReport, error)

type DiplomacyLayout struct {
	Stances       [][]uint32
	Individual    []byte
	Separator     uint32
	AlliedVictory []uint32
	LockTeams     uint8
	ChooseTeams   uint8
	RandomStarts  uint8
	MaxTeams      uint8
}

type DiplomacyMirrorSettings struct {
	Player        int    `json:"player"`
	PlayerLabel   string `json:"player_label"`
	SectionIndex  int    `json:"section_index"`
	IndexBase     string `json:"index_base"`
	Interaction   []int  `json:"interaction,omitempty"`
	AISystem      []int  `json:"ai_system,omitempty"`
	AlliedVictory int    `json:"allied_victory"`
}

type DiplomacyOptionsRecipe struct {
	LockTeams               *bool                 `json:"lock_teams,omitempty"`
	AllowPlayersChooseTeams *bool                 `json:"allow_players_choose_teams,omitempty"`
	RandomStartPoints       *bool                 `json:"random_start_points,omitempty"`
	MaxNumberOfTeams        *int                  `json:"max_number_of_teams,omitempty"`
	AlliedVictory           []AlliedVictoryRecipe `json:"allied_victory,omitempty"`
}

type DiplomacyRecipe struct {
	From   int `json:"from"`
	To     int `json:"to"`
	Stance int `json:"stance"`
}

type DiplomacySettings struct {
	Matrix                  [][]int `json:"matrix,omitempty"`
	AlliedVictory           []bool  `json:"allied_victory,omitempty"`
	LockTeams               bool    `json:"lock_teams"`
	AllowPlayersChooseTeams bool    `json:"allow_players_choose_teams"`
	RandomStartPoints       bool    `json:"random_start_points"`
	MaxNumberOfTeams        int     `json:"max_number_of_teams"`
}

type DisplayTechniques struct {
	ColorMarkup             MarkupColorTechnique    `json:"color_markup"`
	VariableSubstitutionHUD VariableHUDTechnique    `json:"variable_substitution_hud"`
	CreateKillDisplayLoops  CreateKillLoopTechnique `json:"create_kill_display_loops"`
	CaptionObjects          CaptionTechnique        `json:"caption_overhead_label_objects"`
}

type EconomySignal struct {
	Name       string `json:"name"`
	Confidence string `json:"confidence"`
	Count      int    `json:"count,omitempty"`
	Evidence   string `json:"evidence,omitempty"`
}

type EffectDiffItem struct {
	Ordinal       int                         `json:"ordinal"`
	BeforeTrigger int                         `json:"before_trigger"`
	AfterTrigger  int                         `json:"after_trigger"`
	BeforeEffect  int                         `json:"before_effect"`
	AfterEffect   int                         `json:"after_effect"`
	Fields        map[string]EffectFieldDelta `json:"fields,omitempty"`
	Before        *EffectSummary              `json:"before,omitempty"`
	After         *EffectSummary              `json:"after,omitempty"`
}

type EffectDiffReport struct {
	Before        string           `json:"before"`
	After         string           `json:"after"`
	Verification  string           `json:"verification"`
	Query         string           `json:"query"`
	TriggerIndex  *int             `json:"trigger_index,omitempty"`
	BeforeMatches int              `json:"before_matches"`
	AfterMatches  int              `json:"after_matches"`
	Compared      int              `json:"compared"`
	Modified      int              `json:"modified"`
	Added         int              `json:"added"`
	Removed       int              `json:"removed"`
	Changes       []EffectDiffItem `json:"changes,omitempty"`
}

func DiffEffectsFile(beforePath, afterPath string, opts EffectWhereOptions) (EffectDiffReport, error)
    DiffEffectsFile compares matching effects in occurrence order. It is a
    control-oriented diagnostic: trigger/effect indexes are reported, but field
    drift is the primary result because two authors may organize triggers
    differently while expressing the same mechanic.

type EffectFieldDelta struct {
	Before any `json:"before,omitempty"`
	After  any `json:"after,omitempty"`
}

type EffectRecipe struct {
	Op                         string   `json:"op"`
	Message                    string   `json:"message,omitempty"`
	ObjectListUnitID           *int     `json:"object_list_unit_id,omitempty"`
	ObjectListUnitID2          *int     `json:"object_list_unit_id_2,omitempty"`
	SourcePlayer               *int     `json:"source_player,omitempty"`
	TargetPlayer               *int     `json:"target_player,omitempty"`
	TriggerID                  *int     `json:"trigger_id,omitempty"`
	Technology                 *int     `json:"technology,omitempty"`
	Diplomacy                  *int     `json:"diplomacy,omitempty"`
	LocationX                  *int     `json:"location_x,omitempty"`
	LocationY                  *int     `json:"location_y,omitempty"`
	LocationObjectReference    *int     `json:"location_object_reference,omitempty"`
	AreaX1                     *int     `json:"area_x1,omitempty"`
	AreaY1                     *int     `json:"area_y1,omitempty"`
	AreaX2                     *int     `json:"area_x2,omitempty"`
	AreaY2                     *int     `json:"area_y2,omitempty"`
	ObjectGroup                *int     `json:"object_group,omitempty"`
	ObjectType                 *int     `json:"object_type,omitempty"`
	Facet2                     *int     `json:"facet2,omitempty"`
	ActionType                 *int     `json:"action_type,omitempty"`
	AttackStance               *int     `json:"attack_stance,omitempty"`
	MaxUnitsAffected           *int     `json:"max_units_affected,omitempty"`
	IssueGroupCommand          *int     `json:"issue_group_command,omitempty"`
	QueueAction                *int     `json:"queue_action,omitempty"`
	DisableGarrisonUnloadSound *int     `json:"disable_garrison_unload_sound,omitempty"`
	DisplayTime                *int     `json:"display_time,omitempty"`
	InstructionPanelPosition   *int     `json:"instruction_panel_position,omitempty"`
	PlaySound                  *int     `json:"play_sound,omitempty"`
	SoundName                  string   `json:"sound_name,omitempty"`
	UseTagColorForIcon         *int     `json:"use_tag_color_for_icon,omitempty"`
	ItemID                     *int     `json:"item_id,omitempty"`
	Quantity                   *int     `json:"quantity,omitempty"`
	QuantityFloat              *float64 `json:"quantity_float,omitempty"`
	Operation                  *int     `json:"operation,omitempty"`
	OperationName              string   `json:"operation_name,omitempty"`
	TributeList                *int     `json:"tribute_list,omitempty"`
	Resource                   *int     `json:"resource,omitempty"`
	Resource1                  *int     `json:"resource_1,omitempty"`
	Resource1Quantity          *int     `json:"resource_1_quantity,omitempty"`
	Resource2                  *int     `json:"resource_2,omitempty"`
	Resource2Quantity          *int     `json:"resource_2_quantity,omitempty"`
	Resource3                  *int     `json:"resource_3,omitempty"`
	Resource3Quantity          *int     `json:"resource_3_quantity,omitempty"`
	StringID                   *int     `json:"string_id,omitempty"`
	ForceResearchTechnology    *int     `json:"force_research_technology,omitempty"`
	VisibilityState            *int     `json:"visibility_state,omitempty"`
	Scroll                     *int     `json:"scroll,omitempty"`
	FlashObject                *int     `json:"flash_object,omitempty"`
	ObjectAttributes           *int     `json:"object_attributes,omitempty"`
	ObjectState                *int     `json:"object_state,omitempty"`
	Facet                      *int     `json:"facet,omitempty"`
	DisableSound               *int     `json:"disable_sound,omitempty"`
	Enabled                    *int     `json:"enabled,omitempty"`
	ButtonLocation             *int     `json:"button_location,omitempty"`
	Hotkey                     *int     `json:"hotkey,omitempty"`
	TrainTime                  *int     `json:"train_time,omitempty"`
	LocalTechnology            *int     `json:"local_technology,omitempty"`
	PlayerColor                *int     `json:"player_color,omitempty"`
	GlobalSound                *int     `json:"global_sound,omitempty"`
	MutualDiplomacy            *int     `json:"mutual_diplomacy,omitempty"`
	ObjectFilter               *int     `json:"object_filter,omitempty"`
	TimeUnit                   *int     `json:"time_unit,omitempty"`
	TimerID                    *int     `json:"timer_id,omitempty"`
	ResetTimer                 *int     `json:"reset_timer,omitempty"`
	Variable                   *int     `json:"variable,omitempty"`
	Variable2                  *int     `json:"variable2,omitempty"`
	XSFunction                 string   `json:"xs_function,omitempty"`
	SelectedObjectIDs          []int    `json:"selected_object_ids,omitempty"`
}

type EffectSummary struct {
	TriggerIndex      int            `json:"trigger_index,omitempty"`
	TriggerName       string         `json:"trigger_name,omitempty"`
	EffectIndex       int            `json:"effect_index,omitempty"`
	Type              int            `json:"type"`
	TypeName          string         `json:"type_name"`
	UnitConst         int            `json:"unit_const,omitempty"`
	SourcePlayer      int            `json:"source_player,omitempty"`
	TargetPlayer      int            `json:"target_player,omitempty"`
	Location          []int          `json:"location,omitempty"`
	Area              []int          `json:"area,omitempty"`
	TargetTrigger     int            `json:"target_trigger,omitempty"`
	StringID          int            `json:"string_id,omitempty"`
	Text              string         `json:"text,omitempty"`
	DisplayTime       int            `json:"display_time,omitempty"`
	TimeUnit          int            `json:"time_unit,omitempty"`
	TimerID           int            `json:"timer_id,omitempty"`
	ResetTimer        int            `json:"reset_timer,omitempty"`
	Sound             string         `json:"sound,omitempty"`
	Variable          int            `json:"variable,omitempty"`
	Variable2         int            `json:"variable2,omitempty"`
	Operation         int            `json:"operation,omitempty"`
	Quantity          int            `json:"quantity,omitempty"`
	QuantityFloat     float64        `json:"quantity_float,omitempty"`
	Technology        int            `json:"technology,omitempty"`
	Diplomacy         int            `json:"diplomacy,omitempty"`
	Resource          int            `json:"resource,omitempty"`
	ResourceQuantity  int            `json:"resource_quantity,omitempty"`
	ObjectAttribute   int            `json:"object_attribute,omitempty"`
	SelectedObjectIDs []int          `json:"selected_object_ids,omitempty"`
	RawFields         []int          `json:"raw_fields,omitempty"`
	KnownFields       map[string]any `json:"known_fields,omitempty"`
}

type EffectTextEntry struct {
	TriggerIndex int    `json:"trigger_index"`
	TriggerName  string `json:"trigger_name,omitempty"`
	EffectIndex  int    `json:"effect_index"`
	Type         int    `json:"type"`
	StringID     int    `json:"string_id,omitempty"`
	Text         string `json:"text"`
}

type EffectTypeBucket struct {
	Type     int             `json:"type"`
	TypeName string          `json:"type_name"`
	Count    int             `json:"count"`
	Sample   []EffectSummary `json:"sample,omitempty"`
}

type EffectTypeSummary struct {
	TypeName string `json:"type_name"`
	Count    int    `json:"count"`
}

type EffectWhereOptions struct {
	Query            string `json:"query,omitempty"`
	Limit            int    `json:"limit,omitempty"`
	TriggerIndex     *int   `json:"trigger_index,omitempty"`
	IncludeRawFields bool   `json:"include_raw_fields,omitempty"`
	MaxInflatedBytes int    `json:"max_inflated_bytes,omitempty"`
}

type EffectWhereReport struct {
	Path            string          `json:"path,omitempty"`
	Version         string          `json:"version"`
	Verification    string          `json:"verification"`
	HeaderBytes     int             `json:"header_bytes"`
	CompressedBytes int             `json:"compressed_body_bytes"`
	InflatedBytes   int             `json:"inflated_body_bytes"`
	Query           string          `json:"query"`
	Limit           int             `json:"limit"`
	TotalMatches    int             `json:"total_matches"`
	Matches         []EffectSummary `json:"matches,omitempty"`
}

func EffectsWhereFile(path string, opts EffectWhereOptions) (EffectWhereReport, error)

type EffectsCensusOptions struct {
	MaxInflatedBytes int
}

type EffectsCensusReport struct {
	Path                 string             `json:"path,omitempty"`
	Version              string             `json:"version"`
	Verification         string             `json:"verification"`
	HeaderBytes          int                `json:"header_bytes"`
	CompressedBytes      int                `json:"compressed_body_bytes"`
	InflatedBytes        int                `json:"inflated_body_bytes"`
	ParsedThroughSection string             `json:"parsed_through_section"`
	TriggerCount         int                `json:"trigger_count"`
	EffectCount          int                `json:"effect_count"`
	Types                []EffectTypeBucket `json:"types"`
}

func EffectsCensus(data []byte) (EffectsCensusReport, error)

func EffectsCensusFile(path string) (EffectsCensusReport, error)

func EffectsCensusFileWithOptions(path string, opts EffectsCensusOptions) (EffectsCensusReport, error)

func EffectsCensusWithOptions(data []byte, opts EffectsCensusOptions) (EffectsCensusReport, error)

type EffectsOptions struct {
	IncludeRawFields bool
}

type EffectsReport struct {
	Path         string             `json:"path,omitempty"`
	Version      string             `json:"version"`
	Verification string             `json:"verification"`
	Total        int                `json:"total"`
	Types        []EffectTypeBucket `json:"types"`
}

type Encoding string

const (
	EncodingBytes    Encoding = "bytes"
	EncodingCString4 Encoding = "c4"
	EncodingString16 Encoding = "string16"
	EncodingString32 Encoding = "string32"
	EncodingU8       Encoding = "u8"
	EncodingS8       Encoding = "s8"
	EncodingU16      Encoding = "u16"
	EncodingS16      Encoding = "s16"
	EncodingU32      Encoding = "u32"
	EncodingS32      Encoding = "s32"
	EncodingF32      Encoding = "f32"
)
type Evidence struct {
	Tier   string
	Source string
}

type FieldQueryOptions struct {
	Strict bool
}

type FieldQueryResult struct {
	File    string `json:"file"`
	Path    string `json:"path"`
	Version string `json:"version,omitempty"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
	Value   any    `json:"value,omitempty"`
	Hex     string `json:"hex,omitempty"`
	Exists  bool   `json:"exists"`
}
    FieldQueryResult is one stable-path lookup result for scen field.

func QueryField(fieldPath, input string) ([]FieldQueryResult, error)
    QueryField searches one scenario file or every scenario below root. Results
    are sorted by file path so corpus queries are deterministic.

func QueryFieldWithOptions(fieldPath, input string, opts FieldQueryOptions) ([]FieldQueryResult, error)

type FieldRef struct {
	Section string
	Name    string
}

type FieldSpec struct {
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	Repeat     int             `json:"repeat"`
	IsList     *bool           `json:"is_list"`
	Default    json.RawMessage `json:"default"`
	RepeatRule *RepeatRule     `json:"repeat_rule,omitempty"`
}

type File struct {
	Path            string        `json:"path,omitempty"`
	Version         string        `json:"version"`
	PlayerCount     int           `json:"player_count"`
	HeaderBytes     int           `json:"header_bytes"`
	CompressedBytes int           `json:"compressed_body_bytes"`
	InflatedBytes   int           `json:"inflated_body_bytes"`
	Sections        []SectionInfo `json:"sections,omitempty"`
	Triggers        *TriggerInfo  `json:"triggers,omitempty"`
	Units           *UnitInfo     `json:"units,omitempty"`
	Map             *MapInfo      `json:"map,omitempty"`
	Players         []PlayerInfo  `json:"players,omitempty"`
	AI              []AIFileInfo  `json:"ai,omitempty"`

	// Has unexported fields.
}

func Open(path string) (*File, error)

func OpenWithOptions(path string, opts ParseOptions) (*File, error)

func Parse(data []byte) (*File, error)

func ParseWithOptions(data []byte, opts ParseOptions) (*File, error)

func (f *File) AddCreateObjectTrigger(recipe TriggerRecipe) error

func (f *File) AddDisplayInstructionsTrigger(recipe TriggerRecipe) error

func (f *File) AddString(recipe StringRecipe) (int, error)

func (f *File) AddTrigger(recipe TriggerRecipe) error

func (f *File) AddUnit(recipe UnitRecipe) error

func (f *File) AddUnits(recipes []UnitRecipe) error

func (f *File) AddVariable(recipe VariableRecipe) error

func (f *File) Analyze() AnalysisReport

func (f *File) ApplyRecipe(recipe Recipe) error

func (f *File) ApplyTerrainGrid(recipe MapRecipe) error

func (f *File) ClearString(recipe StringRecipe) error

func (f *File) ClearTriggers() error

func (f *File) ClusterScatterUnits(recipe UnitRecipe) ([]UnitRecipe, error)

func (f *File) CopyTerrainArea(recipe MapRecipe) error

func (f *File) CopyTrigger(recipe TriggerRecipe) error

func (f *File) CopyUnitsInArea(recipe UnitRecipe) (int, error)

func (f *File) DeletePlan(request DeletePlanRequest) (DeletePlanReport, error)

func (f *File) DeployCheck(deployTree string) (DeployCheckReport, error)

func (f *File) DeployCheckWithOptions(deployTree string, opts DeployCheckOptions) (DeployCheckReport, error)

func (f *File) DeployXS(opts XSDeployOptions) (XSDeployReport, error)

func (f *File) Describe(opts DescribeOptions) Description

func (f *File) DumpTriggersWithConditions() ([]TriggerConditionDump, error)
    DumpTriggersWithConditions walks trigger_data and decodes every condition's
    known fields alongside the existing effect summaries.

func (f *File) EditTrigger(recipe TriggerRecipe) error

func (f *File) EditUnit(recipe UnitRecipe) error

func (f *File) EditUnitsInArea(recipe UnitRecipe) (int, error)

func (f *File) EditVariable(recipe VariableRecipe) error

func (f *File) Effects() EffectsReport

func (f *File) EffectsWithOptions(opts EffectsOptions) EffectsReport

func (f *File) ErodeTerrain(recipe MapRecipe) error

func (f *File) ErodeTerrainWithMasks(recipe MapRecipe, masks map[string]terrainMask, semantic bool) error

func (f *File) GuessRegions() RegionGuessReport

func (f *File) Idioms() IdiomReport

func (f *File) LayeredCrossfadeTerrain(recipe MapRecipe) error

func (f *File) LayeredCrossfadeTerrainWithMasks(recipe MapRecipe, masks map[string]terrainMask) error

func (f *File) Lint() LintReport

func (f *File) LintWithOptions(opts LintOptions) LintReport

func (f *File) MoveUnitsInArea(recipe UnitRecipe) (int, error)

func (f *File) NoiseFillTerrain(recipe MapRecipe) error

func (f *File) NoiseFillTerrainWithMasks(recipe MapRecipe, masks map[string]terrainMask) error

func (f *File) PaletteUsage(palette datfile.PaletteReport) PaletteUsageReport

func (f *File) Plan(recipe Recipe) (Plan, error)

func (f *File) PreservedDocument() PreservedDocument
    PreservedDocument returns the lossless carrier used by Kit-owned codecs.
    Section migration can replace the temporary parser locator without changing
    the preservation contract.

func (f *File) RawTriggerSection() (RawTriggerSection, error)

func (f *File) RebuildBody() ([]byte, error)

func (f *File) References(opts ReferenceOptions) (ReferenceReport, error)

func (f *File) RemoveSystemPrefix(prefix string) error

func (f *File) RemoveTrigger(recipe TriggerRecipe) error

func (f *File) RemoveTriggers(recipe TriggerRecipe) error

func (f *File) RemoveUnit(recipe UnitRecipe) error

func (f *File) RemoveUnitsForPlayer(recipe UnitRecipe) (int, error)

func (f *File) RemoveUnitsInArea(recipe UnitRecipe) (int, error)

func (f *File) RemoveVariable(recipe VariableRecipe) error

func (f *File) ResizeMap(width, height int) error

func (f *File) SemanticDocument() SemanticDocument
    SemanticDocument projects the current parser result into the Kit-owned
    semantic vocabulary. Until all sections have migrated, this is an adapter;
    its contract is independent of the legacy field representation.

func (f *File) SetDiplomacy(recipe DiplomacyRecipe) error

func (f *File) SetDiplomacyOptions(recipe DiplomacyOptionsRecipe) error

func (f *File) SetGlobalVictory(recipe VictoryRecipe) error

func (f *File) SetPlayer(recipe PlayerRecipe) error

func (f *File) SetResources(recipe ResourceRecipe) error

func (f *File) SetScenario(recipe ScenarioRecipe) error

func (f *File) SetString(recipe StringRecipe) error

func (f *File) SetTerrain(recipe MapRecipe) error

func (f *File) SetTerrainRect(recipe MapRecipe) error

func (f *File) SetTerrainWithMasks(recipe MapRecipe, masks map[string]terrainMask) error

func (f *File) SetXS(recipe XSRecipe) error

func (f *File) SetXSCarrier(recipe XSRecipe, content string) error

func (f *File) SetXSInlineRuntime(recipe XSRecipe, content string) error

func (f *File) Settings() SettingsReport

func (f *File) StringContainsDisconnectRecipe(text string) ([]int, ReferenceReport, Recipe, error)

func (f *File) StringDisconnectRecipe(stringID int) (ReferenceReport, Recipe, error)

func (f *File) StringPrefixDisconnectRecipe(prefix string) ([]int, ReferenceReport, Recipe, error)

func (f *File) StringTextDisconnectRecipe(text string) (int, ReferenceReport, Recipe, error)

func (f *File) Strings() StringReport

func (f *File) Terrain(opts TerrainOptions) (TerrainReport, error)

func (f *File) TombstoneString(recipe StringRecipe) error

func (f *File) TombstoneTrigger(recipe TriggerRecipe) error

func (f *File) TombstoneVariable(recipe VariableRecipe) error

func (f *File) TriggerContainsDisconnectRecipe(text string) ([]int, ReferenceReport, Recipe, error)

func (f *File) TriggerDisconnectRecipe(targetID int) (ReferenceReport, Recipe, error)

func (f *File) TriggerGraph() (*triggergraph.Graph, error)

func (f *File) TriggerNameDisconnectRecipe(name string) (int, ReferenceReport, Recipe, error)

func (f *File) TriggerPrefixDisconnectRecipe(prefix string) ([]int, ReferenceReport, Recipe, error)

func (f *File) UnitCaptionContainsDisconnectRecipe(text string, targetPlayer *int) ([]int, ReferenceReport, Recipe, error)

func (f *File) UnitCaptionDisconnectRecipe(caption string, targetPlayer *int) (int, ReferenceReport, Recipe, error)

func (f *File) UnitCaptionPrefixDisconnectRecipe(prefix string, targetPlayer *int) ([]int, ReferenceReport, Recipe, error)

func (f *File) UnitDisconnectRecipe(referenceID int) (ReferenceReport, Recipe, error)

func (f *File) UnitTypeDisconnectRecipe(unitConst int, targetPlayer *int) ([]int, ReferenceReport, Recipe, error)

func (f *File) UnitsAreaDisconnectRecipe(request DeletePlanRequest) (ReferenceReport, Recipe, error)

func (f *File) UnitsPlayerDisconnectRecipe(player int) ([]int, ReferenceReport, Recipe, error)

func (f *File) VariableContainsDisconnectRecipe(text string) ([]int, ReferenceReport, Recipe, error)

func (f *File) VariableDisconnectRecipe(variableID int) (ReferenceReport, Recipe, error)

func (f *File) VariableNameDisconnectRecipe(name string) (int, ReferenceReport, Recipe, error)

func (f *File) VariablePrefixDisconnectRecipe(prefix string) ([]int, ReferenceReport, Recipe, error)

func (f *File) VerifyRebuild() error

func (f *File) Write(path string) error

func (f *File) WriteReencoded(path string) error
    WriteReencoded serializes the parsed model instead of reusing the original
    compressed bytes. It is primarily useful for round-trip validation and for
    callers that need a freshly encoded scenario after a read-only inspection.

func (f *File) XSAttachment() XSDeployAttachment

func (f *File) XSAttachmentContent() string

func (f *File) XSCensus(opts XSCensusOptions) (XSCensusReport, error)

type GlobalVictoryLayout struct {
	Values [11]uint32
}

type IdiomFinding struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Confidence string   `json:"confidence"`
	Evidence   string   `json:"evidence"`
	Count      int      `json:"count,omitempty"`
	Samples    []string `json:"samples,omitempty"`
}

type IdiomReport struct {
	Path         string         `json:"path,omitempty"`
	Version      string         `json:"version"`
	Verification string         `json:"verification"`
	Count        int            `json:"count"`
	Idioms       []IdiomFinding `json:"idioms,omitempty"`
}

type ImportTriggersOptions struct {
	Select []string `json:"select,omitempty"`
	Prefix string   `json:"prefix,omitempty"`
	Out    string   `json:"-"`
}
    ImportTriggersOptions controls a raw-preserving trigger import. Selection
    entries are trigger indexes, inclusive ranges, or exact trigger names.

type IntCount struct {
	ID    int `json:"id"`
	Count int `json:"count"`
}

type Layout struct {
	Name         string
	Revisions    []LayoutRevision
	Sections     []LayoutSection
	MirrorGroups []MirrorGroup
}
    Layout is Kit's owned description of byte facts. It deliberately does not
    mirror the legacy field JSON: semantic names, evidence, and preservation
    policy are separate from the byte grammar.

type LayoutDocument struct {
	Raw      PreservedDocument
	Semantic SemanticDocument
	Triggers RawTriggerSection
}
    LayoutDocument is the migration seam between the three layers: byte
    preservation, semantic values, and raw-preserving variable sections.

func OpenLayoutDocument(path string) (LayoutDocument, error)
    OpenLayoutDocument builds the owned document model while the legacy parser
    still supplies temporary section locations. No bytes are discarded, and the
    trigger codec does not interpret unknown record fields.

func (d LayoutDocument) Write(path string) error

type LayoutField struct {
	Name     string
	Encoding Encoding
	Count    int
	Opaque   bool
	Evidence Evidence
}

type LayoutRevision struct {
	Name     string
	Versions []string
	Status   string
}

type LayoutSection struct {
	Name   string
	Fields []LayoutField
}

type LintIssue struct {
	Severity     string `json:"severity"`
	Code         string `json:"code"`
	Checkpoint   string `json:"checkpoint,omitempty"`
	Message      string `json:"message"`
	Why          string `json:"why,omitempty"`
	Fix          string `json:"fix,omitempty"`
	FactID       string `json:"fact_id,omitempty"`
	FactTier     string `json:"fact_tier,omitempty"`
	VerifiedDate string `json:"verified_date,omitempty"`
	FixtureRef   string `json:"fixture_ref,omitempty"`
}

type LintOptions struct {
	IncludeProvisional bool
	LoadSafety         bool
	SemanticInvariants bool
	// DatPath enables DAT-backed unit availability checks. Without it the
	// scenario linter reports that this rule was skipped rather than guessing.
	DatPath string
	// Generated marks output produced by Kit. Generated artifacts are held to
	// stricter semantic checks than foreign, hand-authored scenarios.
	Generated bool
}

type LintReport struct {
	Path         string                 `json:"path,omitempty"`
	OK           bool                   `json:"ok"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Issues       []LintIssue            `json:"issues,omitempty"`
	Summary      LintSummary            `json:"summary"`
}

func LintFile(path string) (LintReport, error)

func LintFileWithOptions(path string, opts LintOptions) (LintReport, error)

type LintSummary struct {
	Triggers int `json:"triggers"`
	Units    int `json:"units"`
	Players  int `json:"players"`
	AIFiles  int `json:"ai_files"`
}

type MapInfo struct {
	Width              int                    `json:"width"`
	Height             int                    `json:"height"`
	TileCount          int                    `json:"tile_count"`
	TerrainCounts      map[int]int            `json:"terrain_counts"`
	SecondaryGameModes SecondaryGameModesInfo `json:"secondary_game_modes"`
}

func MapFile(path string) (*MapInfo, error)
    MapFile reads map metadata without requiring later sections such as Units or
    Triggers to be understood.

type MapLayout struct {
	StringStarter1     [2]byte
	WaterDefinition    string
	StringStarter2     [2]byte
	MapColorMood       string
	StringStarter3     [2]byte
	ScriptName         string
	CollideAndCorrect  uint8
	VillagerForceDrop  uint8
	InitialPlayerViews [16][2]int32
	LockCoopAlliances  uint8
	PopulationCaps     [16]uint32
	SecondaryGameModes uint32
	Unknown5           [4]byte
	Unknown4           [4]byte
	NoWavesOnShore     int8
	Width              int32
	Height             int32
	Terrain            []TerrainCell
	Raw                []byte
}

func DecodeMapLayout(data []byte) (MapLayout, int, error)

type MapRecipe struct {
	Op         string   `json:"op"`
	Mask       string   `json:"mask,omitempty"`
	X1         int      `json:"x1"`
	Y1         int      `json:"y1"`
	X2         int      `json:"x2"`
	Y2         int      `json:"y2"`
	Width      *int     `json:"width,omitempty"`
	Height     *int     `json:"height,omitempty"`
	TargetX    *int     `json:"target_x,omitempty"`
	TargetY    *int     `json:"target_y,omitempty"`
	Radius     *int     `json:"radius,omitempty"`
	Thickness  *int     `json:"thickness,omitempty"`
	TerrainID  *int     `json:"terrain_id,omitempty"`
	TerrainID2 *int     `json:"terrain_id_2,omitempty"`
	Elevation  *int     `json:"elevation,omitempty"`
	Layer      *int     `json:"layer,omitempty"`
	Seed       *int     `json:"seed,omitempty"`
	Scale      *float64 `json:"scale,omitempty"`
	Threshold  *float64 `json:"threshold,omitempty"`
	Iterations *int     `json:"iterations,omitempty"`
	TerrainIDs []int    `json:"terrain_ids,omitempty"`
	GridFile   string   `json:"grid_file,omitempty"`

	TerrainIDGrid []int `json:"-"`
	LayerGrid     []int `json:"-"`
	ElevationGrid []int `json:"-"`
	// Has unexported fields.
}

func (recipe MapRecipe) MarshalJSON() ([]byte, error)

func (recipe *MapRecipe) UnmarshalJSON(data []byte) error

type MapSettings struct {
	WaterDefinition string `json:"water_definition"`
}

type MarkupColorTechnique struct {
	Detected bool           `json:"detected"`
	Counts   map[string]int `json:"counts,omitempty"`
}

type MarkupSummary struct {
	ColorTags         map[string]int `json:"color_tags"`
	VariableRefs      []string       `json:"variable_refs"`
	VariableRefCounts map[string]int `json:"variable_ref_counts"`
}

type MaskCellRecipe struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type MaskRecipe struct {
	Name       string           `json:"name"`
	Op         string           `json:"op"`
	X1         int              `json:"x1,omitempty"`
	Y1         int              `json:"y1,omitempty"`
	X2         int              `json:"x2,omitempty"`
	Y2         int              `json:"y2,omitempty"`
	Radius     *int             `json:"radius,omitempty"`
	TerrainIDs []int            `json:"terrain_ids,omitempty"`
	Masks      []string         `json:"masks,omitempty"`
	Seed       *int             `json:"seed,omitempty"`
	Scale      *float64         `json:"scale,omitempty"`
	Threshold  *float64         `json:"threshold,omitempty"`
	Iterations *int             `json:"iterations,omitempty"`
	Invert     *bool            `json:"invert,omitempty"`
	Cells      []MaskCellRecipe `json:"cells,omitempty"`
}

type MechanicOptions struct {
	Kind             string `json:"kind"`
	Grep             string `json:"grep,omitempty"`
	MaxInflatedBytes int    `json:"max_inflated_bytes,omitempty"`
}

type MechanicReport struct {
	Path         string            `json:"path,omitempty"`
	Version      string            `json:"version"`
	Verification string            `json:"verification"`
	Kind         string            `json:"kind"`
	Summary      MechanicSummary   `json:"summary"`
	Triggers     []MechanicTrigger `json:"triggers,omitempty"`
	Warnings     []string          `json:"warnings,omitempty"`
}

func MechanicFile(path string, opts MechanicOptions) (MechanicReport, error)

type MechanicSummary struct {
	TriggerCount   int            `json:"trigger_count"`
	EffectTypes    map[string]int `json:"effect_types,omitempty"`
	ConditionTypes map[string]int `json:"condition_types,omitempty"`
	Players        []int          `json:"players,omitempty"`
}

type MechanicTrigger struct {
	Index      int      `json:"index"`
	Name       string   `json:"name,omitempty"`
	Role       string   `json:"role"`
	Players    []int    `json:"players,omitempty"`
	Effects    []string `json:"effects,omitempty"`
	Conditions []string `json:"conditions,omitempty"`
	Notes      []string `json:"notes,omitempty"`
}

type MessageSlotSettings struct {
	Literal           string `json:"literal,omitempty"`
	StringTableID     int    `json:"string_table_id"`
	UsesStringTableID bool   `json:"uses_string_table_id"`
	EffectiveSource   string `json:"effective_source"`
}

type MessagesSettings struct {
	Instructions MessageSlotSettings `json:"instructions"`
	Hints        MessageSlotSettings `json:"hints"`
	Victory      MessageSlotSettings `json:"victory"`
	Loss         MessageSlotSettings `json:"loss"`
	History      MessageSlotSettings `json:"history"`
	Scouts       MessageSlotSettings `json:"scouts"`
}

type MirrorGroup struct {
	Name          string
	Authoritative string
	Derived       []string
	Policy        string
}
    MirrorGroup makes duplicated storage explicit. The writer policy is not
    implied by field names: callers must choose the authoritative copy.

type NamedSection struct {
	Name string
	Spec SectionSpec
}

type NodeDump struct {
	Name     string     `json:"name"`
	Start    int        `json:"start"`
	End      int        `json:"end"`
	Value    any        `json:"value,omitempty"`
	Fields   []NodeDump `json:"fields,omitempty"`
	Elements []NodeDump `json:"elements,omitempty"`
}

type ObjectImportMapping struct {
	SourceID int    `json:"source_id"`
	TargetID int    `json:"target_id"`
	Label    string `json:"label,omitempty"`
}

type OpaqueSpan struct {
	Name   string
	Offset int
	Length int
}
    OpaqueSpan identifies bytes that the semantic model does not own. Spans are
    retained in their original order and copied verbatim by the preservation
    layer; a dark field is not permission to normalize it.

type OptionsSettings struct {
	FullTechTree              bool                 `json:"full_tech_tree"`
	CollideAndCorrecting      bool                 `json:"collide_and_correcting"`
	VillagerForceDrop         bool                 `json:"villager_force_drop"`
	LockCoopAlliances         bool                 `json:"lock_coop_alliances"`
	TriggerExecutionOrder     int                  `json:"trigger_execution_order"`
	TriggerExecutionOrderName string               `json:"trigger_execution_order_name"`
	InitialPlayerViews        []PlayerViewSettings `json:"initial_player_views,omitempty"`
}

type PaletteUntappedSummary struct {
	UnitID                int    `json:"unit_id"`
	UnitName              string `json:"unit_name"`
	StandingGraphic1      int    `json:"standing_graphic_1"`
	GraphicName           string `json:"graphic_name,omitempty"`
	AvailableVariantCount int    `json:"available_variant_count"`
	UsedVariantCount      int    `json:"used_variant_count"`
	UnusedVariantCount    int    `json:"unused_variant_count"`
	Placements            int    `json:"placements"`
}

type PaletteUsageReport struct {
	Path         string              `json:"path,omitempty"`
	DatPath      string              `json:"dat_path,omitempty"`
	UnitTypes    int                 `json:"unit_types"`
	Placements   int                 `json:"placements"`
	Summary      PaletteUsageSummary `json:"summary"`
	Verification string              `json:"verification"`
	Rows         []PaletteUsageRow   `json:"rows"`
}

func PaletteUsageFile(path, datPath string) (PaletteUsageReport, error)

type PaletteUsageRow struct {
	UnitID                int       `json:"unit_id"`
	UnitName              string    `json:"unit_name"`
	Placements            int       `json:"placements"`
	UnitType              int       `json:"unit_type"`
	UnitClass             int16     `json:"unit_class"`
	UnitClassName         string    `json:"unit_class_name,omitempty"`
	StandingGraphic1      int       `json:"standing_graphic_1"`
	GraphicName           string    `json:"graphic_name,omitempty"`
	FileName              string    `json:"file_name,omitempty"`
	SLP                   int32     `json:"slp"`
	AngleCount            int       `json:"angle_count"`
	FrameCount            int       `json:"frame_count"`
	SequenceType          uint8     `json:"sequence_type"`
	Classification        string    `json:"classification"`
	Confidence            string    `json:"confidence"`
	RotationEncoding      string    `json:"rotation_encoding,omitempty"`
	AvailableVariantCount *int      `json:"available_variant_count,omitempty"`
	UsedIndices           []int     `json:"used_indices,omitempty"`
	UnusedIndices         []int     `json:"unused_indices,omitempty"`
	RawRotations          []float64 `json:"raw_rotations,omitempty"`
	Note                  string    `json:"note,omitempty"`
}

type PaletteUsageSummary struct {
	MultiVariantTypeCount int                      `json:"multi_variant_type_count"`
	ArtworksAvailable     int                      `json:"artworks_available"`
	ArtworksUsed          int                      `json:"artworks_used"`
	BiggestUntapped       []PaletteUntappedSummary `json:"biggest_untapped,omitempty"`
}

type ParseOptions struct {
	MaxInflatedBytes int
	// StopBeforeSection, when set, halts body parsing when the named section is
	// reached and returns the sections parsed so far. Used to read content
	// (units/terrain) from scenarios whose later sections (e.g. Triggers) use a
	// format kit cannot yet fully decode — Microsoft campaign scenarios.
	StopBeforeSection string
}

type PatchReport struct {
	Input               string                 `json:"input"`
	Output              string                 `json:"output"`
	TriggerCountBefore  int                    `json:"trigger_count_before"`
	TriggerCountAfter   int                    `json:"trigger_count_after"`
	UnitCountBefore     int                    `json:"unit_count_before,omitempty"`
	UnitCountAfter      int                    `json:"unit_count_after,omitempty"`
	MapTilesChanged     int                    `json:"map_tiles_changed,omitempty"`
	TimestampOfLastSave int                    `json:"timestamp_of_last_save,omitempty"`
	RebuildOK           bool                   `json:"rebuild_ok"`
	InvariantOK         bool                   `json:"invariant_ok"`
	Verification        aoe2.VerificationClaim `json:"verification"`
}

func PatchRecipeFile(input, output string, recipe Recipe) (PatchReport, error)

func PatchSmokeRecipeFile(input, output string, opts SmokeRecipeOptions) (PatchReport, error)

type Plan struct {
	TriggerCountBefore int            `json:"trigger_count_before"`
	TriggerCountAfter  int            `json:"trigger_count_after"`
	UnitCountBefore    int            `json:"unit_count_before,omitempty"`
	UnitCountAfter     int            `json:"unit_count_after,omitempty"`
	MapTilesChanged    int            `json:"map_tiles_changed,omitempty"`
	Operations         []PlanOp       `json:"operations"`
	Warnings           []string       `json:"warnings,omitempty"`
	Recipe             map[string]any `json:"recipe,omitempty"`
}

func PlanRecipeFile(input string, recipe Recipe) (Plan, error)

type PlanOp struct {
	Op      string `json:"op"`
	Name    string `json:"name,omitempty"`
	Enabled *bool  `json:"enabled,omitempty"`
	Message string `json:"message,omitempty"`
}

type PlayerCoverageFamily struct {
	Key      string         `json:"key"`
	Players  []int          `json:"players"`
	Triggers []TriggerBrief `json:"triggers"`
	Missing  []int          `json:"missing,omitempty"`
}

type PlayerCoverageIssue struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Family   string `json:"family,omitempty"`
}

type PlayerCoverageOptions struct {
	Players          []int `json:"players,omitempty"`
	MaxInflatedBytes int   `json:"max_inflated_bytes,omitempty"`
}

type PlayerCoverageReport struct {
	Path         string                 `json:"path,omitempty"`
	Version      string                 `json:"version"`
	Verification string                 `json:"verification"`
	Players      []int                  `json:"players"`
	Summary      PlayerCoverageSummary  `json:"summary"`
	Families     []PlayerCoverageFamily `json:"families,omitempty"`
	Issues       []PlayerCoverageIssue  `json:"issues,omitempty"`
}

func PlayerCoverageFile(path string, opts PlayerCoverageOptions) (PlayerCoverageReport, error)

type PlayerCoverageSummary struct {
	TriggerFamilies int `json:"trigger_families"`
	Issues          int `json:"issues"`
}

type PlayerDataLayout struct {
	Active          uint32
	Human           uint32
	StrSign1        uint16
	Civilization    string
	StrSign2        uint16
	ArchitectureSet string
	CityMode        uint32
}

type PlayerInfo struct {
	Player           int                        `json:"player"`
	PlayerLabel      string                     `json:"player_label,omitempty"`
	SectionIndex     int                        `json:"section_index"`
	IndexBase        string                     `json:"index_base,omitempty"`
	FieldReferences  map[string]PlayerReference `json:"field_references,omitempty"`
	Active           bool                       `json:"active"`
	Human            bool                       `json:"human"`
	TribeName        string                     `json:"tribe_name,omitempty"`
	NameStringID     int                        `json:"name_string_id"`
	Civilization     string                     `json:"civilization,omitempty"`
	Architecture     string                     `json:"architecture,omitempty"`
	LockCivilization bool                       `json:"lock_civilization"`
	LockPersonality  bool                       `json:"lock_personality"`
	StartingAge      int                        `json:"starting_age"`
	StartingAgeName  string                     `json:"starting_age_name,omitempty"`
	Color            int                        `json:"color"`
	ColorName        string                     `json:"color_name,omitempty"`
	BasePriority     int                        `json:"base_priority"`
	PopulationLimit  int                        `json:"population_limit"`
	AIName           string                     `json:"ai_name,omitempty"`
	AIType           int                        `json:"ai_type,omitempty"`
}

type PlayerRecipe struct {
	Player           int     `json:"player"`
	Active           *bool   `json:"active,omitempty"`
	Human            *bool   `json:"human,omitempty"`
	TribeName        *string `json:"tribe_name,omitempty"`
	Civilization     *string `json:"civilization,omitempty"`
	LockCivilization *bool   `json:"lock_civilization,omitempty"`
	LockPersonality  *bool   `json:"lock_personality,omitempty"`
	AIName           *string `json:"ai_name,omitempty"`
	AIType           *int    `json:"ai_type,omitempty"`
}

type PlayerReference struct {
	Player       string `json:"player"`
	SectionIndex int    `json:"section_index"`
	IndexBase    string `json:"index_base"`
}
    PlayerReference keeps the editor-facing slot separate from the raw array
    index. Scenario sections do not all use the same player indexing convention.

type PlayerResourceValues struct {
	Player     int `json:"player"`
	Gold       int `json:"gold"`
	Wood       int `json:"wood"`
	Food       int `json:"food"`
	Stone      int `json:"stone"`
	TradeGoods int `json:"trade_goods"`
}

type PlayerSettings struct {
	Player            int                        `json:"player"`
	PlayerLabel       string                     `json:"player_label,omitempty"`
	SectionIndex      int                        `json:"section_index"`
	IndexBase         string                     `json:"index_base,omitempty"`
	FieldReferences   map[string]PlayerReference `json:"field_references,omitempty"`
	Active            bool                       `json:"active"`
	Human             bool                       `json:"human"`
	TribeName         string                     `json:"tribe_name,omitempty"`
	NameStringID      int                        `json:"name_string_id"`
	Civilization      string                     `json:"civilization,omitempty"`
	Architecture      string                     `json:"architecture,omitempty"`
	LockCivilization  bool                       `json:"lock_civilization"`
	LockPersonality   bool                       `json:"lock_personality"`
	StartingAge       int                        `json:"starting_age"`
	StartingAgeName   string                     `json:"starting_age_name,omitempty"`
	Color             int                        `json:"color"`
	ColorName         string                     `json:"color_name,omitempty"`
	BasePriority      int                        `json:"base_priority"`
	PopulationLimit   int                        `json:"population_limit"`
	AIName            string                     `json:"ai_name,omitempty"`
	AIType            int                        `json:"ai_type,omitempty"`
	Resources         PlayerResourceValues       `json:"resources"`
	Diplomacy         []int                      `json:"diplomacy,omitempty"`
	AlliedVictory     bool                       `json:"allied_victory"`
	DisabledTechs     []int                      `json:"disabled_techs,omitempty"`
	DisabledUnits     []int                      `json:"disabled_units,omitempty"`
	DisabledBuildings []int                      `json:"disabled_buildings,omitempty"`
}

type PlayerUnitsInfo struct {
	Player int           `json:"player"`
	Count  int           `json:"count"`
	Units  []UnitSummary `json:"units,omitempty"`
}

type PlayerViewSettings struct {
	Player int  `json:"player"`
	X      int  `json:"x"`
	Y      int  `json:"y"`
	Set    bool `json:"set"`
}

type PointRecipe struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type PreservedDocument struct {
	Header       []byte
	Body         []byte
	OriginalFile []byte
	Opaque       []OpaqueSpan
}
    PreservedDocument is the lossless carrier shared by the codecs. The first
    slice only creates it and reads from it; mutation/writer policies arrive in
    later slices once section framing is migrated.

func NewPreservedDocument(header, body []byte) PreservedDocument

func (d PreservedDocument) RebuildBody() []byte

func (d PreservedDocument) RebuildFile() []byte

func (d PreservedDocument) Write(path string) error
    Write writes the original header/body bytes without recompression or
    normalization. It is the safe writer path until a section is explicitly
    marked dirty by a migrated codec.

type RawTriggerRecord struct {
	Index int
	Bytes []byte
}

type RawTriggerSection struct {
	Prefix   []byte
	Records  []RawTriggerRecord
	Suffix   []byte
	Original []byte
}
    RawTriggerSection is the preservation-first trigger codec. It owns record
    framing and leaves every field inside a record opaque until evidence earns a
    typed decoder.

    WE3_Hautevilles_3 is intentionally parked here: the original/editor-saved
    Rosetta pair proves a five-byte separator after a valid first record,
    tombstone/padding records, and a legacy condition-marker form, but does not
    yet provide a second oracle for the per-effect legacy layout. The original
    and twin are retained in the private workshop under runs/convert-official;
    do not promote guessed semantics. Opaque preservation remains exact.

func (s RawTriggerSection) Rebuild() []byte

func (s RawTriggerSection) ReplaceRecord(index int, payload []byte) error

func (s RawTriggerSection) Unchanged() bool

type Recipe struct {
	DatPath          string                  `json:"dat_path,omitempty"`
	Scenario         *ScenarioRecipe         `json:"scenario,omitempty"`
	XS               *XSRecipe               `json:"xs,omitempty"`
	Victory          *VictoryRecipe          `json:"victory,omitempty"`
	Players          []PlayerRecipe          `json:"players,omitempty"`
	Diplomacy        []DiplomacyRecipe       `json:"diplomacy,omitempty"`
	DiplomacyOptions *DiplomacyOptionsRecipe `json:"diplomacy_options,omitempty"`
	Resources        []ResourceRecipe        `json:"resources,omitempty"`
	Strings          []StringRecipe          `json:"strings,omitempty"`
	Variables        []VariableRecipe        `json:"variables,omitempty"`
	Triggers         []TriggerRecipe         `json:"triggers"`
	Masks            []MaskRecipe            `json:"masks,omitempty"`
	Units            []UnitRecipe            `json:"units,omitempty"`
	Map              []MapRecipe             `json:"map,omitempty"`
}

func LoadRecipe(path string) (Recipe, error)

func SmokeRecipe(opts SmokeRecipeOptions) Recipe

type ReferenceOptions struct {
	Kind     string `json:"kind,omitempty"`
	TargetID *int   `json:"target_id,omitempty"`
}

type ReferenceReport struct {
	Path         string              `json:"path,omitempty"`
	Version      string              `json:"version"`
	Verification string              `json:"verification"`
	Summary      ReferenceSummary    `json:"summary"`
	References   []ScenarioReference `json:"references,omitempty"`
	Warnings     []string            `json:"warnings,omitempty"`
}

func ReferencesFile(path string, opts ReferenceOptions) (ReferenceReport, error)

type ReferenceSummary struct {
	Total  int            `json:"total"`
	ByKind map[string]int `json:"by_kind"`
}

type RegionGuess struct {
	Name       string    `json:"name"`
	Box        []float64 `json:"box"`
	Method     string    `json:"method"`
	Confidence string    `json:"confidence"`
	UnitCount  int       `json:"unit_count,omitempty"`
	Player     int       `json:"player,omitempty"`
}

type RegionGuessReport struct {
	Path     string        `json:"path,omitempty"`
	Regions  []RegionGuess `json:"regions"`
	Warnings []string      `json:"warnings,omitempty"`
}

func GuessRegionsFile(path string) (RegionGuessReport, error)

type RepeatRule struct {
	Kind      RepeatRuleKind
	Fields    []FieldRef
	Threshold float64
	Then      int
	Else      int
}
    RepeatRule is Kit-owned codec logic for data-dependent field counts. It is
    deliberately typed: no foreign expression language is interpreted here.

type RepeatRuleKind uint8

const (
	RepeatFromField RepeatRuleKind = iota + 1
	RepeatFromIndexedField
	RepeatProduct
	RepeatSquareRootLength
	RepeatNonEmpty
	RepeatProductNonEmpty
	RepeatFloatAtLeast
	RepeatRoundedFloatAtLeast
	RepeatTypeListOrValue
)
type ResourceRecipe struct {
	Player     int  `json:"player"`
	Gold       *int `json:"gold,omitempty"`
	Wood       *int `json:"wood,omitempty"`
	Food       *int `json:"food,omitempty"`
	Stone      *int `json:"stone,omitempty"`
	TradeGoods *int `json:"trade_goods,omitempty"`
}

type ScenarioDataSet struct {
	Status     string `json:"status"`
	Source     string `json:"source"`
	Confidence string `json:"confidence"`
	Note       string `json:"note,omitempty"`
}

type ScenarioGlossaryOptions struct {
	Limit            int `json:"limit,omitempty"`
	MaxInflatedBytes int `json:"max_inflated_bytes,omitempty"`
}

type ScenarioGlossaryReport struct {
	Path            string             `json:"path,omitempty"`
	Version         string             `json:"version"`
	Verification    string             `json:"verification"`
	HeaderBytes     int                `json:"header_bytes"`
	CompressedBytes int                `json:"compressed_body_bytes"`
	InflatedBytes   int                `json:"inflated_body_bytes"`
	Limit           int                `json:"limit"`
	TriggerCount    int                `json:"trigger_count"`
	EffectCount     int                `json:"effect_count"`
	VariableCount   int                `json:"variable_count"`
	EffectTypes     []EffectTypeBucket `json:"effect_types"`
	TriggerPrefixes []StringCount      `json:"trigger_prefixes,omitempty"`
	Variables       []VariableEntry    `json:"variables,omitempty"`
	XSCalls         []StringCount      `json:"xs_calls,omitempty"`
	TextSamples     []StringCount      `json:"text_samples,omitempty"`
	UnitIDs         []IntCount         `json:"unit_ids,omitempty"`
	TechnologyIDs   []IntCount         `json:"technology_ids,omitempty"`
	AttributeIDs    []IntCount         `json:"attribute_ids,omitempty"`
}

func ScenarioGlossaryFile(path string, opts ScenarioGlossaryOptions) (ScenarioGlossaryReport, error)

type ScenarioHeader struct {
	Version        string
	HeaderLength   uint32
	Savable        int32
	Timestamp      uint32
	Instructions   string
	PlayerCount    uint32
	UnknownValue   uint32
	UnknownValue2  uint32
	UnknownNumbers []uint32
	CreatorName    string
	TriggerCount   uint32
	Raw            []byte
}
    ScenarioHeader is the first-slice typed view of the uncompressed file
    header. Unknown values remain named opaque rather than being given guessed
    semantics.

func DecodeScenarioHeader(data []byte) (ScenarioHeader, int, error)

type ScenarioMapPreset struct {
	Name      string `json:"name"`
	Size      int    `json:"size"`
	StringID  int    `json:"string_id"`
	StringKey string `json:"string_key"`
}

func ScenarioMapPresetBySize(size int) (ScenarioMapPreset, bool)

func ScenarioMapPresets() []ScenarioMapPreset

type ScenarioRecipe struct {
	PlayerCount         *int `json:"player_count,omitempty"`
	TimestampOfLastSave *int `json:"timestamp_of_last_save,omitempty"`
	// StartingAge applies the editor's per-player starting-age field to all
	// playable slots. It is intentionally one value: a recipe that needs
	// asymmetric ages can use the lower-level scenario authoring API.
	StartingAge *int `json:"starting_age,omitempty"`
}

type ScenarioReference struct {
	Kind            string `json:"kind"`
	TargetID        int    `json:"target_id"`
	TargetLabel     string `json:"target_label,omitempty"`
	SourceKind      string `json:"source_kind"`
	SourcePath      string `json:"source_path"`
	Field           string `json:"field"`
	TriggerIndex    int    `json:"trigger_index,omitempty"`
	TriggerName     string `json:"trigger_name,omitempty"`
	ChildKind       string `json:"child_kind,omitempty"`
	ChildIndex      int    `json:"child_index,omitempty"`
	UnitPlayer      int    `json:"unit_player,omitempty"`
	UnitIndex       int    `json:"unit_index,omitempty"`
	UnitReferenceID int    `json:"unit_reference_id,omitempty"`
}

type ScenarioScale struct {
	MapWidth       int            `json:"map_width"`
	MapHeight      int            `json:"map_height"`
	TriggerCount   int            `json:"trigger_count"`
	EnabledCount   int            `json:"enabled_count"`
	VariableCount  int            `json:"variable_count"`
	UnitTotal      int            `json:"unit_total"`
	UnitsByPlayer  map[int]int    `json:"units_by_player"`
	EffectCount    int            `json:"effect_count"`
	EffectTypeBins map[string]int `json:"effect_type_bins"`
	PlayerCount    int            `json:"player_count"`
	Sections       []SectionInfo  `json:"sections,omitempty"`
}

type ScenarioScanMeta struct {
	Index            int    `json:"index"`
	Name             string `json:"name,omitempty"`
	Description      string `json:"description,omitempty"`
	ShortDescription string `json:"short_description,omitempty"`
	Enabled          uint32 `json:"enabled"`
	Looping          int8   `json:"looping"`
	EffectCount      int    `json:"effect_count"`
	ConditionCount   int    `json:"condition_count"`
}

type ScenarioScanOptions struct {
	MaxInflatedBytes int
}

type SecondaryGameModesInfo struct {
	Bits  int      `json:"bits"`
	Names []string `json:"names,omitempty"`
}
    SecondaryGameModesInfo is the editor's Map.secondary_game_modes bitfield.
    The names are derived from the measured DE bit assignments, not guessed from
    the raw integer.

type SectionDump struct {
	Name   string     `json:"name"`
	Start  int        `json:"start"`
	End    int        `json:"end"`
	Fields []NodeDump `json:"fields,omitempty"`
}

type SectionInfo struct {
	Name  string `json:"name"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

type SectionSpec struct {
	Fields  []FieldSpec
	Structs map[string]SectionSpec
}

type SemanticDiplomacy struct {
	Matrix        [][]int
	AlliedVictory []bool
}

type SemanticDocument struct {
	Version   string
	Players   []SemanticPlayer
	Diplomacy SemanticDiplomacy
	Victory   map[string]int
	Units     []SemanticUnit
}
    SemanticDocument is the editor-facing layer over the byte layout. It keeps
    P1..P8 labels and mirror policies out of byte-field names.

type SemanticPlayer struct {
	Slot          int
	Label         string
	Settings      PlayerSettings
	Diplomacy     []int
	AlliedVictory bool
}

type SemanticUnit struct {
	Player int
	Unit   UnitSummary
}

type SettingsReport struct {
	Path               string                    `json:"path,omitempty"`
	Version            string                    `json:"version"`
	Verification       string                    `json:"verification"`
	PlayerCount        int                       `json:"player_count"`
	Players            []PlayerSettings          `json:"players,omitempty"`
	Options            OptionsSettings           `json:"options"`
	Map                MapSettings               `json:"map"`
	Diplomacy          DiplomacySettings         `json:"diplomacy"`
	DiplomacyMirrors   []DiplomacyMirrorSettings `json:"diplomacy_mirrors,omitempty"`
	Messages           MessagesSettings          `json:"messages"`
	Cinematics         CinematicsSettings        `json:"cinematics"`
	Victory            map[string]int            `json:"victory,omitempty"`
	VictoryModeName    string                    `json:"victory_mode_name,omitempty"`
	SecondaryGameModes SecondaryGameModesInfo    `json:"secondary_game_modes"`
	Resources          []PlayerResourceValues    `json:"resources,omitempty"`
}

func SettingsFile(path string) (SettingsReport, error)

type ShopCatalog struct {
	Name             string            `json:"name,omitempty"`
	Player           int               `json:"player,omitempty"`
	SetupTriggerName string            `json:"setup_trigger_name,omitempty"`
	Cleanup          bool              `json:"cleanup,omitempty"`
	ClearTriggers    bool              `json:"clear_triggers,omitempty"`
	RenameShops      *bool             `json:"rename_shops,omitempty"`
	Area             *ShopCatalogArea  `json:"area,omitempty"`
	Shops            []ShopCatalogShop `json:"shops"`
}

func LoadShopCatalogFile(path string) (ShopCatalog, error)

func ShopCatalogExample() ShopCatalog

type ShopCatalogArea struct {
	X1 int `json:"x1"`
	Y1 int `json:"y1"`
	X2 int `json:"x2"`
	Y2 int `json:"y2"`
}

type ShopCatalogItem struct {
	Label       string                `json:"label"`
	Description string                `json:"description,omitempty"`
	TokenUnit   int                   `json:"token_unit"`
	Button      int                   `json:"button,omitempty"`
	TrainTime   *int                  `json:"train_time,omitempty"`
	Hotkey      *int                  `json:"hotkey,omitempty"`
	Costs       []ShopCatalogItemCost `json:"costs,omitempty"`
}

type ShopCatalogItemCost struct {
	ResourceID *int   `json:"resource_id,omitempty"`
	Resource   string `json:"resource,omitempty"`
	Amount     int    `json:"amount"`
}

type ShopCatalogReport struct {
	Name          string               `json:"name,omitempty"`
	Player        int                  `json:"player"`
	ShopCount     int                  `json:"shop_count"`
	ItemCount     int                  `json:"item_count"`
	Cleanup       bool                 `json:"cleanup"`
	ClearTriggers bool                 `json:"clear_triggers"`
	Recipe        Recipe               `json:"recipe"`
	ShopSummaries []ShopCatalogSummary `json:"shop_summaries"`
	Warnings      []string             `json:"warnings,omitempty"`
	Verification  string               `json:"verification"`
}

func ShopCatalogRecipe(catalog ShopCatalog) (ShopCatalogReport, error)

type ShopCatalogShop struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	ShopUnit    int               `json:"shop_unit"`
	Player      int               `json:"player,omitempty"`
	X           *float64          `json:"x,omitempty"`
	Y           *float64          `json:"y,omitempty"`
	ReferenceID *int              `json:"reference_id,omitempty"`
	Caption     string            `json:"caption,omitempty"`
	Items       []ShopCatalogItem `json:"items"`
}

type ShopCatalogSummary struct {
	Name      string `json:"name"`
	ShopUnit  int    `json:"shop_unit"`
	Player    int    `json:"player"`
	ItemCount int    `json:"item_count"`
	ButtonMin int    `json:"button_min,omitempty"`
	ButtonMax int    `json:"button_max,omitempty"`
	HasUnit   bool   `json:"has_unit"`
}

type SmokeRecipeOptions struct {
	Player       int
	PlayerSet    bool
	UnitConst    int
	UnitConstSet bool
	X            int
	XSet         bool
	Y            int
	YSet         bool
	TerrainID    int
	TerrainIDSet bool
	Elevation    int
	ElevationSet bool
	Layer        int
	LayerSet     bool
}

type Spec struct {
	Sections []NamedSection
}

func LoadCurrentDESpec() (*Spec, error)

func LoadDESpecForVersion(version string) (*Spec, error)

type StringCount struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

type StringRecipe struct {
	Op       string  `json:"op"`
	ID       *int    `json:"id,omitempty"`
	Text     string  `json:"text,omitempty"`
	SetText  *string `json:"set_text,omitempty"`
	OldText  string  `json:"old_text,omitempty"`
	Required *bool   `json:"required,omitempty"`
}

type StringReport struct {
	Path          string             `json:"path,omitempty"`
	Version       string             `json:"version"`
	Verification  string             `json:"verification"`
	Messages      map[string]string  `json:"messages"`
	StringTable   []StringTableEntry `json:"string_table,omitempty"`
	Variables     []VariableEntry    `json:"variables,omitempty"`
	TriggerText   []TriggerTextEntry `json:"trigger_text,omitempty"`
	EffectText    []EffectTextEntry  `json:"effect_text,omitempty"`
	MarkupSummary MarkupSummary      `json:"markup_summary"`
	Warnings      []string           `json:"warnings,omitempty"`
}

func StringsFile(path string) (StringReport, error)
    StringsFile returns the structured string report when the scenario format
    is fully decoded. For a known-but-not-yet-mapped format, it falls back to a
    deliberately partial printable-string extraction. This keeps text recovery
    useful without presenting a guessed section layout as a successful parse.

type StringTableEntry struct {
	ID     int    `json:"id"`
	Source string `json:"source"`
	Text   string `json:"text"`
}

type StructureFieldDiff struct {
	Path         string `json:"path"`
	BeforeHex    string `json:"before_hex,omitempty"`
	AfterHex     string `json:"after_hex,omitempty"`
	BeforeValue  any    `json:"before_value,omitempty"`
	AfterValue   any    `json:"after_value,omitempty"`
	BeforeStart  int    `json:"before_start,omitempty"`
	AfterStart   int    `json:"after_start,omitempty"`
	BeforeBytes  int    `json:"before_bytes"`
	AfterBytes   int    `json:"after_bytes"`
	BeforeExists bool   `json:"before_exists"`
	AfterExists  bool   `json:"after_exists"`
}
    StructureFieldDiff is a path-aligned field change. Offsets are diagnostic
    context only; paths remain stable when a preceding structure grows.

type SuppressedRun struct {
	Kind   string `json:"kind"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
	Detail string `json:"detail,omitempty"`
}

type TerrainAggregate struct {
	TerrainID        int                `json:"terrain_id"`
	Count            int                `json:"count"`
	Bounds           TerrainBounds      `json:"bounds"`
	Centroid         TerrainPoint       `json:"centroid"`
	Components       int                `json:"components"`
	LargestComponent int                `json:"largest_component"`
	ComponentBounds  []TerrainComponent `json:"component_bounds,omitempty"`
}

type TerrainBounds struct {
	MinX int `json:"min_x"`
	MinY int `json:"min_y"`
	MaxX int `json:"max_x"`
	MaxY int `json:"max_y"`
}

type TerrainCell struct {
	TerrainID uint8
	Elevation uint8
	Unused    [3]byte
	Layer     int16
}

type TerrainComponent struct {
	Index    int           `json:"index"`
	Count    int           `json:"count"`
	Bounds   TerrainBounds `json:"bounds"`
	Centroid TerrainPoint  `json:"centroid"`
}

type TerrainFilter struct {
	ID int `json:"id"`
}

type TerrainOptions struct {
	ID           *int
	IncludeTiles bool
}

type TerrainPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type TerrainReport struct {
	Path         string             `json:"path,omitempty"`
	Width        int                `json:"width"`
	Height       int                `json:"height"`
	TileCount    int                `json:"tile_count"`
	Returned     int                `json:"returned"`
	Filter       *TerrainFilter     `json:"filter,omitempty"`
	Aggregates   []TerrainAggregate `json:"aggregates"`
	Tiles        []TerrainTile      `json:"tiles,omitempty"`
	Verification string             `json:"verification"`
}

func TerrainFile(path string, opts TerrainOptions) (TerrainReport, error)

type TerrainTile struct {
	X         int    `json:"x"`
	Y         int    `json:"y"`
	TerrainID int    `json:"terrain_id"`
	Elevation int    `json:"elevation"`
	Layer     int    `json:"layer"`
	UnusedHex string `json:"unused_hex,omitempty"`
}

type TriggerBrief struct {
	Index   int    `json:"index"`
	Name    string `json:"name,omitempty"`
	Enabled uint32 `json:"enabled"`
	Looping int8   `json:"looping"`
}

type TriggerConditionDump struct {
	Index      int              `json:"index"`
	Name       string           `json:"name"`
	Enabled    uint32           `json:"enabled"`
	Looping    int8             `json:"looping"`
	Conditions []map[string]any `json:"conditions"`
	Effects    []EffectSummary  `json:"effects"`
}
    TriggerConditionDump is a full per-trigger view including decoded
    conditions, which TriggerInfo summarizes away.

type TriggerConditionsOptions struct {
	Limit            int `json:"limit,omitempty"`
	MaxInflatedBytes int `json:"max_inflated_bytes,omitempty"`
}

type TriggerConditionsReport struct {
	Path            string                 `json:"path,omitempty"`
	Version         string                 `json:"version"`
	Verification    string                 `json:"verification"`
	HeaderBytes     int                    `json:"header_bytes"`
	CompressedBytes int                    `json:"compressed_body_bytes"`
	InflatedBytes   int                    `json:"inflated_body_bytes"`
	Limit           int                    `json:"limit,omitempty"`
	TotalTriggers   int                    `json:"total_triggers"`
	Triggers        []TriggerConditionDump `json:"triggers,omitempty"`
}

func TriggerConditionsFile(path string, opts TriggerConditionsOptions) (TriggerConditionsReport, error)

type TriggerFlowOptions struct {
	Grep             string `json:"grep,omitempty"`
	UnitRef          *int   `json:"unit_ref,omitempty"`
	Variable         *int   `json:"variable,omitempty"`
	Limit            int    `json:"limit,omitempty"`
	MaxInflatedBytes int    `json:"max_inflated_bytes,omitempty"`
}

type TriggerFlowReport struct {
	Path         string             `json:"path,omitempty"`
	Version      string             `json:"version"`
	Verification string             `json:"verification"`
	Query        TriggerFlowOptions `json:"query"`
	Stages       []TriggerFlowStage `json:"stages,omitempty"`
	Warnings     []string           `json:"warnings,omitempty"`
}

func TriggerFlowFile(path string, opts TriggerFlowOptions) (TriggerFlowReport, error)

type TriggerFlowStage struct {
	Stage    string            `json:"stage"`
	Triggers []MechanicTrigger `json:"triggers,omitempty"`
}

type TriggerImportMapping struct {
	SourceID int    `json:"source_id"`
	TargetID int    `json:"target_id"`
	Name     string `json:"name,omitempty"`
}

type TriggerImportReport struct {
	Source                 string                  `json:"source"`
	Destination            string                  `json:"destination"`
	Output                 string                  `json:"output,omitempty"`
	Copied                 []TriggerImportMapping  `json:"copied,omitempty"`
	Triggers               []TriggerImportMapping  `json:"trigger_id_map,omitempty"`
	Variables              []VariableImportMapping `json:"variable_id_map,omitempty"`
	Objects                []ObjectImportMapping   `json:"object_id_map,omitempty"`
	Unresolved             []string                `json:"unresolved,omitempty"`
	Verification           string                  `json:"verification"`
	DestinationCountBefore int                     `json:"destination_trigger_count_before"`
	DestinationCountAfter  int                     `json:"destination_trigger_count_after"`
}

func ImportTriggersFile(sourcePath, destinationPath string, opts ImportTriggersOptions) (TriggerImportReport, error)
    ImportTriggersFile copies selected typed trigger records from source into
    destination. The source bytes are parsed and changed only for references and
    the optional name prefix; destination records and all unrelated sections are
    otherwise left to the normal preservation writer.

type TriggerInfo struct {
	Version       float64          `json:"version"`
	Count         int              `json:"count"`
	GraphSHA256   string           `json:"graph_sha256,omitempty"`
	DisplayOrder  []uint32         `json:"display_order,omitempty"`
	Variables     int              `json:"variables"`
	RecordStart   int              `json:"record_start"`
	RecordEnd     int              `json:"record_end"`
	Triggers      []TriggerSummary `json:"triggers,omitempty"`
	InvariantOK   bool             `json:"invariant_ok"`
	InvariantNote string           `json:"invariant_note,omitempty"`
}

type TriggerNeighborhoodEdge struct {
	From        int    `json:"from"`
	To          int    `json:"to"`
	Kind        string `json:"kind"`
	SourceChild string `json:"source_child,omitempty"`
}

type TriggerNeighborhoodNode struct {
	Index      int      `json:"index"`
	Name       string   `json:"name,omitempty"`
	Depth      int      `json:"depth"`
	Enabled    uint32   `json:"enabled"`
	Looping    int8     `json:"looping"`
	Effects    int      `json:"effects"`
	Conditions int      `json:"conditions"`
	Reasons    []string `json:"reasons,omitempty"`
}

type TriggerNeighborhoodOptions struct {
	TriggerIndex     *int `json:"trigger_index,omitempty"`
	UnitRef          *int `json:"unit_ref,omitempty"`
	Variable         *int `json:"variable,omitempty"`
	Depth            int  `json:"depth,omitempty"`
	Limit            int  `json:"limit,omitempty"`
	MaxInflatedBytes int  `json:"max_inflated_bytes,omitempty"`
}

type TriggerNeighborhoodReport struct {
	Path         string                     `json:"path,omitempty"`
	Version      string                     `json:"version"`
	Verification string                     `json:"verification"`
	Query        TriggerNeighborhoodOptions `json:"query"`
	RootTriggers []int                      `json:"root_triggers,omitempty"`
	Nodes        []TriggerNeighborhoodNode  `json:"nodes,omitempty"`
	Edges        []TriggerNeighborhoodEdge  `json:"edges,omitempty"`
	Warnings     []string                   `json:"warnings,omitempty"`
}

func TriggerNeighborhoodFile(path string, opts TriggerNeighborhoodOptions) (TriggerNeighborhoodReport, error)

type TriggerRecipe struct {
	Op                        string            `json:"op"`
	Name                      string            `json:"name"`
	Message                   string            `json:"message"`
	Description               string            `json:"description,omitempty"`
	ShortDescription          string            `json:"short_description,omitempty"`
	TargetIndex               *int              `json:"target_index,omitempty"`
	TargetIndexes             []int             `json:"target_indexes,omitempty"`
	TargetName                string            `json:"target_name,omitempty"`
	TargetPrefix              string            `json:"target_prefix,omitempty"`
	SetName                   *string           `json:"set_name,omitempty"`
	DescriptionStringID       *int              `json:"description_string_table_id,omitempty"`
	ShortDescriptionStringID  *int              `json:"short_description_string_table_id,omitempty"`
	DisplayAsObjective        *bool             `json:"display_as_objective,omitempty"`
	DisplayOnScreen           *bool             `json:"display_on_screen,omitempty"`
	MakeHeader                *bool             `json:"make_header,omitempty"`
	MuteObjectives            *bool             `json:"mute_objectives,omitempty"`
	ExecuteOnLoad             *bool             `json:"execute_on_load,omitempty"`
	ObjectiveDescriptionOrder *int              `json:"objective_description_order,omitempty"`
	Enabled                   *bool             `json:"enabled,omitempty"`
	Looping                   *bool             `json:"looping,omitempty"`
	RemoveEffects             []int             `json:"remove_effects,omitempty"`
	RemoveConditions          []int             `json:"remove_conditions,omitempty"`
	ClearEffects              *bool             `json:"clear_effects,omitempty"`
	ClearConditions           *bool             `json:"clear_conditions,omitempty"`
	ReplaceEffects            []EffectRecipe    `json:"replace_effects,omitempty"`
	ReplaceConditions         []ConditionRecipe `json:"replace_conditions,omitempty"`
	Effects                   []EffectRecipe    `json:"effects,omitempty"`
	Conditions                []ConditionRecipe `json:"conditions,omitempty"`
	DisplayTime               *int              `json:"display_time,omitempty"`
	InstructionPanelPosition  *int              `json:"instruction_panel_position,omitempty"`
	SourcePlayer              *int              `json:"source_player,omitempty"`
	PlaySound                 *int              `json:"play_sound,omitempty"`
	UseTagColorForIcon        *int              `json:"use_tag_color_for_icon,omitempty"`
	ObjectListUnitID          *int              `json:"object_list_unit_id,omitempty"`
	LocationX                 *int              `json:"location_x,omitempty"`
	LocationY                 *int              `json:"location_y,omitempty"`
	ItemID                    *int              `json:"item_id,omitempty"`
	Facet                     *int              `json:"facet,omitempty"`
	DisableSound              *int              `json:"disable_sound,omitempty"`
}

type TriggerSearchMatch struct {
	TriggerIndex     int               `json:"trigger_index"`
	TriggerName      string            `json:"trigger_name,omitempty"`
	Description      string            `json:"description,omitempty"`
	ShortDescription string            `json:"short_description,omitempty"`
	Enabled          uint32            `json:"enabled"`
	Looping          int8              `json:"looping"`
	EffectCount      int               `json:"effect_count"`
	ConditionCount   int               `json:"condition_count"`
	EffectIndex      *int              `json:"effect_index,omitempty"`
	EffectType       int               `json:"effect_type,omitempty"`
	EffectTypeName   string            `json:"effect_type_name,omitempty"`
	ConditionIndex   *int              `json:"condition_index,omitempty"`
	ConditionType    int               `json:"condition_type,omitempty"`
	ConditionName    string            `json:"condition_name,omitempty"`
	MatchedFields    map[string]string `json:"matched_fields"`
}

type TriggerSearchOptions struct {
	Grep             string `json:"grep,omitempty"`
	EffectQuery      string `json:"effect_query,omitempty"`
	Message          string `json:"message,omitempty"`
	ConditionQuery   string `json:"condition_query,omitempty"`
	UnitRef          *int   `json:"unit_ref,omitempty"`
	UnitType         *int   `json:"unit_type,omitempty"`
	Player           *int   `json:"player,omitempty"`
	Variable         *int   `json:"variable,omitempty"`
	Area             []int  `json:"area,omitempty"`
	Limit            int    `json:"limit,omitempty"`
	MaxInflatedBytes int    `json:"max_inflated_bytes,omitempty"`
}

type TriggerSearchReport struct {
	Path            string               `json:"path,omitempty"`
	Version         string               `json:"version"`
	Verification    string               `json:"verification"`
	HeaderBytes     int                  `json:"header_bytes"`
	CompressedBytes int                  `json:"compressed_body_bytes"`
	InflatedBytes   int                  `json:"inflated_body_bytes"`
	Limit           int                  `json:"limit"`
	Query           TriggerSearchOptions `json:"query"`
	TotalMatches    int                  `json:"total_matches"`
	Matches         []TriggerSearchMatch `json:"matches,omitempty"`
}

func TriggerSearchFile(path string, opts TriggerSearchOptions) (TriggerSearchReport, error)

type TriggerSummary struct {
	Index                         int                `json:"index"`
	Name                          string             `json:"name"`
	Enabled                       uint32             `json:"enabled"`
	Looping                       int8               `json:"looping"`
	ExecuteOnLoad                 int                `json:"execute_on_load"`
	DescriptionStringTableID      int                `json:"description_string_table_id"`
	DisplayAsObjective            int                `json:"display_as_objective"`
	ObjectiveDescriptionOrder     uint32             `json:"objective_description_order"`
	MakeHeader                    int                `json:"make_header"`
	ShortDescriptionStringTableID int                `json:"short_description_string_table_id"`
	DisplayOnScreen               int                `json:"display_on_screen"`
	MuteObjectives                int                `json:"mute_objectives"`
	Effects                       int                `json:"effects"`
	EffectData                    []EffectSummary    `json:"effect_data,omitempty"`
	Conditions                    int                `json:"conditions"`
	ConditionData                 []ConditionSummary `json:"condition_data,omitempty"`
	RecordStart                   int                `json:"record_start"`
	RecordEnd                     int                `json:"record_end"`
}

type TriggerTextEntry struct {
	TriggerIndex int    `json:"trigger_index"`
	TriggerName  string `json:"trigger_name,omitempty"`
	Field        string `json:"field"`
	Text         string `json:"text"`
}

type UnitDiffSummary struct {
	ReferenceID int     `json:"reference_id"`
	UnitConst   int     `json:"unit_const"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
}

type UnitInfo struct {
	NumberOfUnitSections int               `json:"number_of_unit_sections"`
	NumberOfPlayers      int               `json:"number_of_players"`
	Sections             []PlayerUnitsInfo `json:"sections"`
	Total                int               `json:"total"`
}

type UnitRecipe struct {
	Op                    string        `json:"op"`
	Player                int           `json:"player"`
	UnitConst             int           `json:"unit_const"`
	X                     *float64      `json:"x,omitempty"`
	Y                     *float64      `json:"y,omitempty"`
	Z                     *float64      `json:"z,omitempty"`
	ReferenceID           *int          `json:"reference_id,omitempty"`
	TargetPlayer          *int          `json:"target_player,omitempty"`
	TargetIndex           *int          `json:"target_index,omitempty"`
	TargetCaption         string        `json:"target_caption,omitempty"`
	TargetUnitConst       *int          `json:"target_unit_const,omitempty"`
	TargetAreaX1          *float64      `json:"target_area_x1,omitempty"`
	TargetAreaY1          *float64      `json:"target_area_y1,omitempty"`
	TargetAreaX2          *float64      `json:"target_area_x2,omitempty"`
	TargetAreaY2          *float64      `json:"target_area_y2,omitempty"`
	TargetX               *float64      `json:"target_x,omitempty"`
	TargetY               *float64      `json:"target_y,omitempty"`
	OffsetX               *float64      `json:"offset_x,omitempty"`
	OffsetY               *float64      `json:"offset_y,omitempty"`
	ReferenceIDBase       *int          `json:"reference_id_base,omitempty"`
	Count                 *int          `json:"count,omitempty"`
	Centers               []PointRecipe `json:"centers,omitempty"`
	ClusterShape          string        `json:"cluster_shape,omitempty"`
	Snap                  string        `json:"snap,omitempty"`
	Distribution          string        `json:"distribution,omitempty"`
	Jitter                *float64      `json:"jitter,omitempty"`
	Spread                *float64      `json:"spread,omitempty"`
	MinDistance           *float64      `json:"min_distance,omitempty"`
	AllowedTerrainIDs     []int         `json:"allowed_terrain_ids,omitempty"`
	Exact                 *bool         `json:"exact,omitempty"`
	Seed                  *int          `json:"seed,omitempty"`
	CaptionSuffix         string        `json:"caption_suffix,omitempty"`
	SetPlayer             *int          `json:"set_player,omitempty"`
	Status                *int          `json:"status,omitempty"`
	Rotation              *float64      `json:"rotation,omitempty"`
	RotationChoices       []float64     `json:"rotation_choices,omitempty"`
	InitialAnimationFrame *int          `json:"initial_animation_frame,omitempty"`
	GarrisonedInID        *int          `json:"garrisoned_in_id,omitempty"`
	CaptionStringID       *int          `json:"caption_string_id,omitempty"`
	CaptionString         string        `json:"caption_string,omitempty"`
}

type UnitSummary struct {
	Index           int     `json:"index"`
	ReferenceID     int     `json:"reference_id"`
	UnitConst       int     `json:"unit_const"`
	X               float64 `json:"x"`
	Y               float64 `json:"y"`
	Z               float64 `json:"z"`
	Rotation        float64 `json:"rotation"`
	Status          int     `json:"status"`
	CaptionStringID int     `json:"caption_string_id,omitempty"`
	CaptionString   string  `json:"caption_string,omitempty"`
}

type VariableEntry struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type VariableHUDTechnique struct {
	Detected bool     `json:"detected"`
	Refs     []string `json:"refs,omitempty"`
	Count    int      `json:"count"`
}

type VariableImportMapping struct {
	SourceID int    `json:"source_id"`
	TargetID int    `json:"target_id"`
	Name     string `json:"name"`
	Action   string `json:"action"` // reused or added
}

type VariableRecipe struct {
	Op         string  `json:"op"`
	ID         *int    `json:"id,omitempty"`
	Name       string  `json:"name,omitempty"`
	TargetID   *int    `json:"target_id,omitempty"`
	TargetName string  `json:"target_name,omitempty"`
	SetName    *string `json:"set_name,omitempty"`
}

type VictoryRecipe struct {
	ConquestRequired               *int `json:"conquest_required,omitempty"`
	Ruins                          *int `json:"ruins,omitempty"`
	ArtifactsRequired              *int `json:"artifacts_required,omitempty"`
	Discovery                      *int `json:"discovery,omitempty"`
	ExploredPercentOfMapRequired   *int `json:"explored_percent_of_map_required,omitempty"`
	GoldRequired                   *int `json:"gold_required,omitempty"`
	AllCustomConditionsRequired    *int `json:"all_custom_conditions_required,omitempty"`
	Mode                           *int `json:"mode,omitempty"`
	RequiredScoreForScoreVictory   *int `json:"required_score_for_score_victory,omitempty"`
	TimeForTimedGameIn10thsOfAYear *int `json:"time_for_timed_game_in_10ths_of_a_year,omitempty"`
}

type WatermarkSignal struct {
	Kind       string   `json:"kind"`
	Confidence string   `json:"confidence"`
	Terms      []string `json:"terms,omitempty"`
	Triggers   []string `json:"triggers,omitempty"`
}

type XSCensusCounts struct {
	ScriptContentAttachments int `json:"script_content_attachments"`
	EmbeddedCarriers         int `json:"embedded_carriers"`
	ScriptCalls              int `json:"script_calls"`
	UniqueCalledFunctions    int `json:"unique_called_functions"`
	Functions                int `json:"functions"`
	Includes                 int `json:"includes"`
	ResolvedIncludes         int `json:"resolved_includes"`
	MissingIncludes          int `json:"missing_includes"`
	Declarations             int `json:"declarations"`
	CrossFileNonExtern       int `json:"cross_file_nonextern"`
	Findings                 int `json:"findings"`
}

type XSCensusOptions struct {
	DeployTree          string
	CheckCreateResults  bool
	DatPath             string
	RequireXSCheck      bool
	XSCheckPath         string
	AllowMissingXSCheck bool
}

type XSCensusReport struct {
	Path                  string                   `json:"path,omitempty"`
	Version               string                   `json:"version,omitempty"`
	Verification          string                   `json:"verification"`
	OK                    bool                     `json:"ok"`
	XS                    XSDeployAttachment       `json:"xs"`
	ScriptContentEmbedded bool                     `json:"script_content_embedded"`
	EmbeddedCarriers      []XSEmbeddedCarrier      `json:"embedded_carriers,omitempty"`
	DeployTree            string                   `json:"deploy_tree,omitempty"`
	ScriptCalls           []XSScriptCall           `json:"script_calls,omitempty"`
	UniqueCalledFunctions []string                 `json:"unique_called_functions,omitempty"`
	Functions             []xsauthor.XSFunction    `json:"functions,omitempty"`
	Analysis              *xsauthor.AnalysisReport `json:"analysis,omitempty"`
	Findings              []DeployCheckFinding     `json:"findings,omitempty"`
	Counts                XSCensusCounts           `json:"counts"`
}

func XSCensusFile(path string, opts XSCensusOptions) (XSCensusReport, error)

type XSDeployAttachment struct {
	ScriptName              string `json:"script_name,omitempty"`
	ScriptFilePath          string `json:"script_file_path,omitempty"`
	ScriptFileContentBytes  int    `json:"script_file_content_bytes,omitempty"`
	ScriptFileContentSHA256 string `json:"script_file_content_sha256,omitempty"`
	ResolvedPath            string `json:"resolved_path,omitempty"`
	ResolvedRelativePath    string `json:"resolved_relative_path,omitempty"`
}

type XSDeployOptions struct {
	DeployTree   string
	Name         string
	CarrierIndex *int
	TriggerIndex *int
	Force        bool
}

type XSDeployReport struct {
	Path                 string `json:"path,omitempty"`
	DeployTree           string `json:"deploy_tree"`
	Output               string `json:"output"`
	ResolvedRelativePath string `json:"resolved_relative_path"`
	Name                 string `json:"name"`
	Source               string `json:"source"`
	ContentBytes         int    `json:"content_bytes"`
	ContentSHA256        string `json:"content_sha256"`
	Verification         string `json:"verification"`
}

func DeployXSFile(path string, opts XSDeployOptions) (XSDeployReport, error)

type XSEmbeddedCarrier struct {
	TriggerIndex  int    `json:"trigger_index"`
	TriggerName   string `json:"trigger_name,omitempty"`
	EffectIndex   int    `json:"effect_index"`
	Title         string `json:"title,omitempty"`
	SourceBytes   int    `json:"source_bytes"`
	SourceSHA256  string `json:"source_sha256"`
	ContentBytes  int    `json:"content_bytes"`
	ContentSHA256 string `json:"content_sha256"`
	Source        string `json:"-"`
	Content       string `json:"-"`
}

type XSRecipe struct {
	Name                string `json:"name,omitempty"`
	Content             string `json:"content,omitempty"`
	ContentFile         string `json:"content_file,omitempty"`
	Mode                string `json:"mode,omitempty"`
	CarrierTitle        string `json:"carrier_title,omitempty"`
	CarrierTriggerName  string `json:"carrier_trigger_name,omitempty"`
	CarrierTriggerIndex *int   `json:"carrier_trigger_index,omitempty"`
	ReplaceCarrier      *bool  `json:"replace_carrier,omitempty"`
	ClearAttachment     *bool  `json:"clear_attachment,omitempty"`
}

type XSScriptCall struct {
	TriggerIndex int      `json:"trigger_index"`
	TriggerName  string   `json:"trigger_name,omitempty"`
	EffectIndex  int      `json:"effect_index"`
	Message      string   `json:"message"`
	Calls        []string `json:"calls,omitempty"`
}
```

## aoe2kit/pkg/testfixtures

```go
package testfixtures // import "aoe2kit/pkg/testfixtures"


CONSTANTS

const RootEnv = "AOE2KIT_FIXTURE_ROOT"

FUNCTIONS

func Path(t testing.TB, rel string) string
```

## aoe2kit/pkg/text

```go
package text // import "aoe2kit/pkg/text"

Package text plans bounded Gaia text windows without coupling callers to XS
geometry. The standalone examples/text/gaia_text.xs remains the no-toolchain
runtime path; this package is Kit's optional document/cost convenience layer.

TYPES

type Alignment struct {
	Font        string  `json:"font"`
	Text        string  `json:"text"`
	TargetWidth float64 `json:"target_width"`
	TextWidth   float64 `json:"text_width"`
	Padding     float64 `json:"padding"`
}

type Behavior string

const (
	BehaviorStatic     Behavior = "static"
	BehaviorCrawl      Behavior = "crawl"
	BehaviorTypewriter Behavior = "typewriter"
	BehaviorPage       Behavior = "page"
	BehaviorDialogue   Behavior = "dialogue"
)
type Document struct {
	Source string
	Glyphs []int
}

func Compile(source, alphabet string) (Document, error)

type FontMetrics struct {
	Widths             map[string]float64 `json:"widths"`
	SpaceWidth         float64            `json:"space_width"`
	RenderedSpaceWidth float64            `json:"rendered_space_width,omitempty"`
}

type Intent struct {
	Style     Style
	Document  Document
	Behavior  Behavior
	Lifecycle Lifecycle
	Budget    int
	MaxWindow int
}

type Lifecycle string

const (
	LifecyclePermanent      Lifecycle = "permanent"
	LifecycleHold           Lifecycle = "hold"
	LifecycleUntilOffscreen Lifecycle = "until_offscreen"
	LifecycleWhileVariable  Lifecycle = "while_variable"
	LifecycleReadingTime    Lifecycle = "reading_time"
)
type MetricDocument struct {
	Schema string                 `json:"schema"`
	Build  string                 `json:"build"`
	Source string                 `json:"source"`
	Fonts  map[string]FontMetrics `json:"fonts"`
}
    MetricDocument is the auditable data contract for extracted game font
    metrics. Kit does not ship the game's atlases or private extraction tool.

func LoadMetrics(data []byte) (MetricDocument, error)

func (doc MetricDocument) AlignRight(fontName, value string, targetWidth float64) (Alignment, error)

func (doc MetricDocument) GenerateXSWidthHelper(fontName, functionName string) (string, error)
    GenerateXSWidthHelper emits the runtime width function while baking only the
    measured table. Callers choose the invisible-space glyphs separately.

func (doc MetricDocument) Measure(fontName, value string) (float64, error)

type Renderer string

const (
	RendererBitmap Renderer = "bitmap"
	RendererStroke Renderer = "stroke"
)
type Style struct {
	Name               string
	Renderer           Renderer
	ForegroundMaterial int
	BackgroundMaterial int
	ScaleTilesPerCell  float64
	SpriteWidthTiles   float64
	CoverageRatio      float64
	Budget             int
	GlyphCosts         []int
}

func (s Style) SamplingCells() (float64, error)

type Window struct {
	Start      int
	End        int
	Glyphs     []int
	ObjectCost int
	NextCursor int
	Complete   bool
}

func MaterializeWindow(intent Intent, cursor int) (Window, error)
    MaterializeWindow returns the largest document prefix that fits the declared
    object ceiling. A glyph that cannot fit as the first item is an error;
    no request is silently truncated.
```

## aoe2kit/pkg/triggergraph

```go
package triggergraph // import "aoe2kit/pkg/triggergraph"


FUNCTIONS

func ParseLocated(blob []byte) (*Graph, *Region, error)

TYPES

type DiffChange struct {
	Kind         string `json:"kind"`
	Field        string `json:"field,omitempty"`
	TriggerIndex int    `json:"trigger_index,omitempty"`
	TriggerName  string `json:"trigger_name,omitempty"`
	Before       any    `json:"before,omitempty"`
	After        any    `json:"after,omitempty"`
	Detail       string `json:"detail,omitempty"`
}

type DiffCounts struct {
	Triggers   int `json:"triggers"`
	Effects    int `json:"effects"`
	Conditions int `json:"conditions"`
	Messages   int `json:"messages"`
}

type DiffOptions struct {
	Limit int `json:"limit,omitempty"`
}

type DiffReport struct {
	Before       string       `json:"before,omitempty"`
	After        string       `json:"after,omitempty"`
	Verification string       `json:"verification"`
	Same         bool         `json:"same"`
	BeforeSHA256 string       `json:"before_sha256,omitempty"`
	AfterSHA256  string       `json:"after_sha256,omitempty"`
	BeforeCounts DiffCounts   `json:"before_counts"`
	AfterCounts  DiffCounts   `json:"after_counts"`
	Summary      DiffSummary  `json:"summary"`
	Changes      []DiffChange `json:"changes,omitempty"`
	Warnings     []string     `json:"warnings,omitempty"`
}

func Diff(before, after *Graph, opts DiffOptions) *DiffReport

type DiffSummary struct {
	Same        bool `json:"same"`
	Total       int  `json:"total_changes"`
	Shown       int  `json:"shown_changes"`
	Limit       int  `json:"limit,omitempty"`
	Added       int  `json:"added"`
	Removed     int  `json:"removed"`
	Modified    int  `json:"modified"`
	CountFields int  `json:"count_fields"`
}

type Graph struct {
	SHA256         string           `json:"sha256"`
	TriggerCount   int              `json:"trigger_count"`
	EffectCount    int              `json:"effect_count"`
	ConditionCount int              `json:"condition_count"`
	MessageCount   int              `json:"message_count"`
	Start          int              `json:"start"`
	End            int              `json:"end"`
	Warnings       []string         `json:"warnings,omitempty"`
	Triggers       []map[string]any `json:"triggers,omitempty"`
}

func FromTriggers(triggers []map[string]any, start, end int, warnings []string) (*Graph, error)

func Parse(blob []byte, opts ParseOptions) (*Graph, error)

func Scan(blob []byte) (*Graph, error)

type NeighborhoodEdge struct {
	From           int    `json:"from"`
	To             int    `json:"to"`
	Kind           string `json:"kind"`
	EffectIndex    *int   `json:"effect_index,omitempty"`
	ConditionIndex *int   `json:"condition_index,omitempty"`
}

type NeighborhoodNode struct {
	Index          int      `json:"index"`
	Name           string   `json:"name,omitempty"`
	Enabled        int      `json:"enabled,omitempty"`
	Looping        int      `json:"looping,omitempty"`
	EffectTypes    []string `json:"effect_types,omitempty"`
	ConditionTypes []string `json:"condition_types,omitempty"`
	Distance       int      `json:"distance"`
}

type NeighborhoodOptions struct {
	TriggerIndex *int `json:"trigger_index,omitempty"`
	Depth        int  `json:"depth,omitempty"`
	Limit        int  `json:"limit,omitempty"`
}

type NeighborhoodReport struct {
	Verification string              `json:"verification"`
	GraphSHA256  string              `json:"graph_sha256"`
	TriggerCount int                 `json:"trigger_count"`
	Query        NeighborhoodOptions `json:"query"`
	Nodes        []NeighborhoodNode  `json:"nodes"`
	Edges        []NeighborhoodEdge  `json:"edges"`
	Warnings     []string            `json:"warnings,omitempty"`
}

func Neighborhood(graph *Graph, opts NeighborhoodOptions) (*NeighborhoodReport, error)

type ParseOptions struct {
	Start int
	End   int
	Count int
}

type Region struct {
	Start       int      `json:"start"`
	End         int      `json:"end"`
	StringCount int      `json:"string_count"`
	FirstString string   `json:"first_string,omitempty"`
	LastString  string   `json:"last_string,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
}

func FindRegion(blob []byte) (*Region, error)

type ScenarioString struct {
	MarkerOffset int    `json:"marker_offset"`
	EndOffset    int    `json:"end_offset"`
	Text         string `json:"text"`
}

func ScenarioStrings(blob []byte) []ScenarioString

type SearchMatch struct {
	TriggerIndex     int            `json:"trigger_index"`
	TriggerName      string         `json:"trigger_name,omitempty"`
	Description      string         `json:"description,omitempty"`
	ShortDescription string         `json:"short_description,omitempty"`
	Enabled          int            `json:"enabled,omitempty"`
	Looping          int            `json:"looping,omitempty"`
	EffectIndex      *int           `json:"effect_index,omitempty"`
	EffectType       int            `json:"effect_type,omitempty"`
	EffectTypeName   string         `json:"effect_type_name,omitempty"`
	ConditionIndex   *int           `json:"condition_index,omitempty"`
	ConditionType    int            `json:"condition_type,omitempty"`
	ConditionName    string         `json:"condition_name,omitempty"`
	Message          string         `json:"message,omitempty"`
	KnownFields      map[string]any `json:"known_fields,omitempty"`
	MatchedFields    map[string]any `json:"matched_fields,omitempty"`
}

type SearchOptions struct {
	Grep         string `json:"grep,omitempty"`
	Effect       string `json:"effect,omitempty"`
	Condition    string `json:"condition,omitempty"`
	Player       *int   `json:"player,omitempty"`
	Variable     *int   `json:"variable,omitempty"`
	UnitConst    *int   `json:"unit_const,omitempty"`
	TriggerLimit int    `json:"trigger_limit,omitempty"`
	MatchLimit   int    `json:"match_limit,omitempty"`
}

type SearchReport struct {
	Verification   string        `json:"verification"`
	GraphSHA256    string        `json:"graph_sha256"`
	TriggerCount   int           `json:"trigger_count"`
	EffectCount    int           `json:"effect_count"`
	ConditionCount int           `json:"condition_count"`
	Query          SearchOptions `json:"query"`
	TriggersShown  int           `json:"triggers_shown"`
	TotalMatches   int           `json:"total_matches"`
	Matches        []SearchMatch `json:"matches,omitempty"`
	Warnings       []string      `json:"warnings,omitempty"`
}

func Search(graph *Graph, opts SearchOptions) (*SearchReport, error)
```

## aoe2kit/pkg/xs

```go
package xs // import "aoe2kit/pkg/xs"


CONSTANTS

const MaxSidecarBytes = (1 << 20) - 1
    MaxSidecarBytes is the largest sidecar payload the engine can retain for a
    single XS file. Keep this in the shared decoder so generators and readers
    report against the same contract.


FUNCTIONS

func CheckASCIIFile(path string) error
    CheckASCIIFile enforces the byte contract of generated XS. DE's tokenizer
    accepts ASCII comments but rejects non-ASCII bytes in string literals; Kit's
    generators own their output, so rejecting every non-ASCII byte keeps the
    generated/runtime boundary deterministic and avoids a misleading external
    parser pass.

func DiscoverSiblingTypesFile(input string) string
    DiscoverSiblingTypesFile finds the schema belonging to an XS sidecar.
    A matching basename wins; otherwise discovery only succeeds when exactly
    one recognized schema remains. Ambiguous directories return no result rather
    than allowing an unrelated schema to misdecode the run.

func ParseDataTypes(spec string) ([]string, error)
func RequireParser(explicit string) (string, error)
    RequireParser resolves the external checker for generated XS. Generated
    output must never be reported as ready when the independent parser was not
    available; callers that intentionally support foreign or optional XS should
    continue using ResolveParser directly.

func ResolveInclude(fromPath, includePath string, contents map[string]string) (string, bool)
func ResolveParser(explicit string) (string, error)
    ResolveParser finds an explicitly requested checker, AOE2KIT_XS_CHECK,
    the user-local pinned install, or an xs-check executable on PATH. An empty
    result means unavailable.

func SchemaFromTypesDocument(document string) (schema string, ok bool, err error)
    SchemaFromTypesDocument resolves the documented multi-record .types format
    emitted by Kit generators. It intentionally returns false for the older
    comma-separated --types form, which remains handled by ParseDataTypes.

func WriteBridgeOutputs(report BridgeReport, outDir string) error
func WriteDatagenModule(report DatagenReport, output string) error
func WriteShimRecipe(report ShimReport, output string) error
func WriteTypesDocument(path, document string) error
    WriteTypesDocument writes the sibling schema contract used by generated
    XS sidecars. Keeping this operation here prevents generators from silently
    drifting on empty or malformed schema documents.

func LoadExpectedDataLedger(path string) ([]expectedDataValue, error)

TYPES

type AnalysisReport struct {
	Findings      []SourceFinding `json:"findings,omitempty"`
	Entry         string          `json:"entry,omitempty"`
	Files         []string        `json:"files"`
	Includes      []IncludeRef    `json:"includes,omitempty"`
	Declarations  []SymbolDecl    `json:"declarations,omitempty"`
	CrossFileUses []CrossFileUse  `json:"cross_file_uses,omitempty"`
	Warnings      []string        `json:"warnings,omitempty"`
}

func AnalyzeFiles(opts AnalyzeOptions) (AnalysisReport, error)

type AnalyzeOptions struct {
	EntryPath          string
	Files              []SourceFile
	CheckCreateResults bool
	DatPath            string
}

type BridgeIssue struct {
	Severity string `json:"severity"`
	Variable string `json:"variable,omitempty"`
	Message  string `json:"message"`
}

type BridgeReport struct {
	Verification        VerificationClaim `json:"verification"`
	ModuleName          string            `json:"module_name"`
	Variables           []BridgeVariable  `json:"variables"`
	XSModule            string            `json:"xs_module"`
	TriggerVariableDefs []BridgeVariable  `json:"trigger_variable_defs"`
	Issues              []BridgeIssue     `json:"issues,omitempty"`
	OK                  bool              `json:"ok"`
}

func GenerateBridge(spec BridgeSpec) (BridgeReport, error)

type BridgeSpec struct {
	Variables     map[string]int `json:"variables"`
	Reads         []string       `json:"reads,omitempty"`
	Writes        []string       `json:"writes,omitempty"`
	XSReads       []string       `json:"xs_reads,omitempty"`
	XSWrites      []string       `json:"xs_writes,omitempty"`
	TriggerReads  []string       `json:"trigger_reads,omitempty"`
	TriggerWrites []string       `json:"trigger_writes,omitempty"`
	ModuleName    string         `json:"module_name,omitempty"`
}

func LoadBridgeSpec(path string) (BridgeSpec, error)

type BridgeVariable struct {
	Name       string `json:"name"`
	ID         int    `json:"id"`
	XSName     string `json:"xs_name"`
	ConstName  string `json:"const_name"`
	GetterName string `json:"getter_name"`
}

type CrossFileUse struct {
	Name        string `json:"name"`
	DeclaredIn  string `json:"declared_in"`
	DeclaredAt  int    `json:"declared_at"`
	UsedIn      string `json:"used_in"`
	UsedAt      int    `json:"used_at"`
	Finding     string `json:"finding"`
	Recommended string `json:"recommended_fix"`
}

type DataAssertion struct {
	Index    int    `json:"index"`
	Name     string `json:"name,omitempty"`
	Expected string `json:"expected"`
	Actual   string `json:"actual,omitempty"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
}

type DataAssertionCounts struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Unknown int `json:"unknown"`
}

type DataDecodeOptions struct {
	Types  []string
	Ledger string
}

type DataDecodeReport struct {
	Path         string              `json:"path,omitempty"`
	OK           bool                `json:"ok"`
	Verification VerificationClaim   `json:"verification"`
	Decode       DataInspectReport   `json:"decode"`
	Ledger       string              `json:"ledger,omitempty"`
	Assertions   []DataAssertion     `json:"assertions,omitempty"`
	Summary      DataAssertionCounts `json:"summary"`
	Errors       []string            `json:"errors,omitempty"`
}

func DecodeDataFile(path string, opts DataDecodeOptions) (DataDecodeReport, error)

type DataInspectOptions struct {
	Types         []string
	MaxAutoString int
}

type DataInspectReport struct {
	Path              string            `json:"path"`
	SizeBytes         int               `json:"size_bytes"`
	Mode              string            `json:"mode"`
	OK                bool              `json:"ok"`
	Verification      VerificationClaim `json:"verification"`
	Values            []DataValue       `json:"values"`
	RemainingBytes    int               `json:"remaining_bytes,omitempty"`
	RemainingOffset   int               `json:"remaining_offset,omitempty"`
	RemainingHex      string            `json:"remaining_hex,omitempty"`
	Errors            []string          `json:"errors,omitempty"`
	HeuristicWarnings []string          `json:"heuristic_warnings,omitempty"`
}

func InspectDataBytes(data []byte, opts DataInspectOptions) DataInspectReport

func InspectDataFile(path string, opts DataInspectOptions) (DataInspectReport, error)

type DataSchemaDecodeOptions struct {
	Schema string
}

type DataSchemaDecodeReport struct {
	Path         string                  `json:"path,omitempty"`
	OK           bool                    `json:"ok"`
	Schema       string                  `json:"schema"`
	Verification VerificationClaim       `json:"verification"`
	Header       map[string]any          `json:"header,omitempty"`
	Rows         []DataSchemaRow         `json:"rows,omitempty"`
	Footer       map[string]any          `json:"footer,omitempty"`
	Summary      DataSchemaDecodeSummary `json:"summary"`
	Warnings     []string                `json:"warnings,omitempty"`
	Errors       []string                `json:"errors,omitempty"`
}

func DecodeDataBytesSchema(data []byte, opts DataSchemaDecodeOptions) DataSchemaDecodeReport

func DecodeDataFileSchema(path string, opts DataSchemaDecodeOptions) (DataSchemaDecodeReport, error)

type DataSchemaDecodeSummary struct {
	SizeBytes     int            `json:"size_bytes"`
	ByteCap       int            `json:"byte_cap"`
	WithinByteCap bool           `json:"within_byte_cap"`
	BytesConsumed int            `json:"bytes_consumed"`
	Rows          int            `json:"rows"`
	CompleteRows  int            `json:"complete_rows"`
	HasFooter     bool           `json:"has_footer"`
	RemainingFrom int            `json:"remaining_from,omitempty"`
	TagCounts     map[string]int `json:"tag_counts,omitempty"`
}

type DataSchemaRow struct {
	Index   int            `json:"index"`
	Offset  int            `json:"offset"`
	PhaseID int            `json:"phase_id,omitempty"`
	TimeS   int            `json:"time_s,omitempty"`
	Fields  map[string]any `json:"fields"`
}

type DataValue struct {
	Index          int       `json:"index"`
	Offset         int       `json:"offset"`
	Type           string    `json:"type"`
	Size           int       `json:"size"`
	String         string    `json:"string,omitempty"`
	Int            *int32    `json:"int,omitempty"`
	UInt           *uint32   `json:"uint,omitempty"`
	Float          *float32  `json:"float,omitempty"`
	Vector         []float32 `json:"vector,omitempty"`
	RawHex         string    `json:"raw_hex,omitempty"`
	Heuristic      bool      `json:"heuristic,omitempty"`
	FloatCandidate *float32  `json:"float_candidate,omitempty"`
	Note           string    `json:"note,omitempty"`
}

func (value DataValue) MarshalJSON() ([]byte, error)
    MarshalJSON keeps ordinary float fields numeric, but represents NaN and
    infinities as strings so one non-finite engine result cannot invalidate the
    entire diagnostic report. This is an output-only policy; the decoded Float
    pointer still contains the original IEEE value for callers that need it.

type DatagenArray struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	Default json.RawMessage   `json:"default,omitempty"`
	Values  []json.RawMessage `json:"values"`
}

type DatagenReport struct {
	Verification VerificationClaim `json:"verification"`
	Function     string            `json:"function"`
	Arrays       []DatagenSummary  `json:"arrays"`
	XSModule     string            `json:"xs_module"`
	Verified     bool              `json:"verified"`
}

func GenerateDatagen(spec DatagenSpec) (DatagenReport, error)

type DatagenSpec struct {
	Function string         `json:"function"`
	Arrays   []DatagenArray `json:"arrays"`
	Includes []string       `json:"includes,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

func LoadDatagenSpec(path string) (DatagenSpec, error)

type DatagenSummary struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Count  int    `json:"count"`
	Handle string `json:"handle"`
}

type DifferentialFinding struct {
	Class      string `json:"class"`
	File       string `json:"file,omitempty"`
	Line       int    `json:"line,omitempty"`
	Severity   string `json:"severity"`
	KitCode    string `json:"kit_code,omitempty"`
	KitMessage string `json:"kit_message,omitempty"`
	XSCode     string `json:"xscheck_code,omitempty"`
	XSMessage  string `json:"xscheck_message,omitempty"`
}

func CompareFindings(kit []SourceFinding, external []ExternalFinding) []DifferentialFinding

type ExternalFinding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Message  string `json:"message"`
}

func ExternalFindings(report ParseCheckReport) []ExternalFinding

type IncludeRef struct {
	From    string `json:"from"`
	Line    int    `json:"line"`
	Path    string `json:"path"`
	Target  string `json:"target,omitempty"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type ParseCheckReport struct {
	Checker string `json:"checker"`
	Status  string `json:"status"`
	Output  string `json:"output,omitempty"`
}
    ParseCheckReport records the result of an optional external XS parser.
    Kit's source lint remains useful, but it is not a replacement for the game's
    grammar or an external parser that models it.

func CheckFile(path, checker string) (ParseCheckReport, error)
    CheckFile runs an external checker against one XS file. xs-check currently
    exits zero even when it prints parser errors, so output is part of the gate.

type ShimOptions struct {
	Paths             []string
	Prefix            string
	Enabled           bool
	IncludeParamFuncs bool
}

type ShimReport struct {
	Verification VerificationClaim `json:"verification"`
	Functions    []XSFunction      `json:"functions"`
	Emitted      int               `json:"emitted"`
	Skipped      int               `json:"skipped"`
	Recipe       map[string]any    `json:"recipe"`
}

func GenerateShims(opts ShimOptions) (ShimReport, error)

type SourceFile struct {
	Path    string
	Content string
}

func LoadSourceTree(entryPath string) ([]SourceFile, error)

type SourceFinding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	GameLine int    `json:"game_line,omitempty"`
	Message  string `json:"message"`
	Fix      string `json:"fix"`
	Evidence string `json:"evidence"`
}
    SourceFinding is a conservative source check, not an XS compiler verdict.

func LintSource(path, source string) []SourceFinding
    LintSource checks hazards observed in the September engine fixtures.
    Comments and literals are lexed separately so examples in prose cannot
    become code.

func LintSourceWithOptions(path, source string, opts SourceLintOptions) []SourceFinding

type SourceLintOptions struct {
	CheckCreateResults bool
	DatPath            string
}
    SourceLintOptions controls checks that need more context than the historical
    source-only lint. The default LintSource behavior remains compatible for
    callers that lint foreign snippets; Kit generators can opt into strict
    create-result checking.

type SymbolDecl struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Type   string `json:"type"`
	Extern bool   `json:"extern"`
	File   string `json:"file"`
	Line   int    `json:"line"`
}

type VerificationClaim struct {
	Label             string `json:"label"`
	StructureVerified bool   `json:"structure_verified"`
	EngineVerified    bool   `json:"engine_verified"`
	Note              string `json:"note"`
}

func StructureVerifiedClaim(note string) VerificationClaim

type XSFunction struct {
	Name       string `json:"name"`
	ReturnType string `json:"return_type"`
	Params     string `json:"params"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	Emitted    bool   `json:"emitted"`
	SkipReason string `json:"skip_reason,omitempty"`
}

func ScanFunctionSource(path, content string) []XSFunction

func ScanFunctions(paths []string) ([]XSFunction, error)
```

