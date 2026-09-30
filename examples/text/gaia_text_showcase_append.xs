
int showcaseLastSecond = -1;
void main() {
  fontInit(); fontLoadTables();
  drawSentence(0, 84.0, 45.0, "GAIA TEXT", 939, 0.0875);
  drawSentence(1, 84.0, 52.0, "RUNTIME HUD", 939, 0.0875);
  drawSentence(2, 84.0, 59.0, "SCORE", 939, 0.0875);
  drawNumber(3, 84.0, 63.0, 0, 5, 939, 0.0875);
  drawStrokeNumber(4, 84.0, 70.0, 0, 4, 939, 0.125, 0);

  pathMove(0.0, 0.0);
  pathLine(8.0, 0.0);
  pathQuadratic(11.0, 4.0, 8.0, 8.0);
  pathCubic(5.0, 11.0, 2.0, 11.0, 0.0, 8.0);
  pathStroke(5, 64.0, 52.0, 939, 0.0875, 0.35);

  pathMove(0.0, 0.0);
  pathCubic(3.0, -4.0, 7.0, 12.0, 10.0, 8.0);
  pathQuadratic(13.0, 4.0, 16.0, 0.0);
  pathStroke(6, 64.0, 68.0, 939, 0.0875, 0.35);
}

rule showcaseTick
active
minInterval 1
maxInterval 1
{
  int now = xsGetGameTime();
  if (now != showcaseLastSecond) {
    showcaseLastSecond = now;
    setNumber(3, now * 37, 5);
    drawStrokeNumber(4, 84.0, 70.0, now, 4, 939, 0.125, 0);
  }
}
