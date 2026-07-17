/* Lightweight hand-rolled SVG area/line chart — no chart libraries.
 * Ports the CustomPaint graph from overview_page.dart (_GraphPainter) but
 * rendered as an area + line instead of bars. Fixed logical viewBox scaled to
 * the container width via preserveAspectRatio="none"; strokes use
 * vector-effect="non-scaling-stroke" so they stay crisp when stretched. */

const W = 600;
const H = 100;
const GRID_LINES = 5;

export function UsageChart({
  data,
  color = 'var(--color-accent)',
  height = 100,
}: {
  data: number[];
  color?: string;
  height?: number;
}) {
  const points = data.length > 0 ? data : [0];
  const max = points.reduce((m, v) => Math.max(m, v), 0);
  const effectiveMax = max > 0 ? max : 1;

  const stepX = points.length > 1 ? W / (points.length - 1) : W;
  const y = (v: number) => H - (v / effectiveMax) * (H - 4);

  const coords = points.map((v, i) => [i * stepX, y(v)] as const);
  const linePath = coords
    .map(([px, py], i) => `${i === 0 ? 'M' : 'L'} ${px.toFixed(2)} ${py.toFixed(2)}`)
    .join(' ');
  const areaPath = `${linePath} L ${W} ${H} L 0 ${H} Z`;
  const gradientId = `usage-grad-${Math.round(color.length * 7 + points.length)}`;

  return (
    <svg
      width="100%"
      height={height}
      viewBox={`0 0 ${W} ${H}`}
      preserveAspectRatio="none"
      role="img"
      aria-label="Usage over time"
      className="block"
    >
      <defs>
        <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={color} stopOpacity={0.28} />
          <stop offset="100%" stopColor={color} stopOpacity={0} />
        </linearGradient>
      </defs>

      {Array.from({ length: GRID_LINES }, (_, i) => {
        const gy = (H * i) / (GRID_LINES - 1);
        return (
          <line
            key={i}
            x1={0}
            y1={gy}
            x2={W}
            y2={gy}
            stroke="var(--border)"
            strokeWidth={1}
            strokeOpacity={0.5}
            vectorEffect="non-scaling-stroke"
          />
        );
      })}

      {max > 0 && <path d={areaPath} fill={`url(#${gradientId})`} />}
      <path
        d={linePath}
        fill="none"
        stroke={color}
        strokeWidth={2}
        strokeLinejoin="round"
        strokeLinecap="round"
        vectorEffect="non-scaling-stroke"
      />
    </svg>
  );
}
