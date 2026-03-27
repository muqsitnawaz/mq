import React, {type CSSProperties} from 'react';
import {AbsoluteFill, Easing, interpolate, spring, useCurrentFrame, useVideoConfig} from 'remotion';

const palette = {
  bg: '#07111f',
  bgAlt: '#0d1830',
  text: '#f5f7fb',
  textMuted: '#95a8c7',
  cyan: '#79f2ff',
  cyanSoft: '#15394a',
  amber: '#ffb453',
  amberSoft: '#382719',
  lime: '#b7ff83',
  border: 'rgba(149, 168, 199, 0.18)',
};

const sceneStyle: CSSProperties = {
  padding: '88px 104px',
  display: 'flex',
  flexDirection: 'column',
  justifyContent: 'center',
  color: palette.text,
};

const surfaceStyle: CSSProperties = {
  background: 'rgba(7, 17, 31, 0.72)',
  border: `1px solid ${palette.border}`,
  boxShadow: '0 40px 120px rgba(0, 0, 0, 0.35)',
  backdropFilter: 'blur(28px)',
};

const terminalStyle: CSSProperties = {
  ...surfaceStyle,
  borderRadius: 30,
  padding: '28px 32px',
  fontFamily: '"SF Mono", "Menlo", "Monaco", monospace',
};

const fadeRange = (frame: number, start: number, end: number) =>
  interpolate(frame, [start, start + 15, end - 15, end], [0, 1, 1, 0], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
  });

const rise = (frame: number, start: number, distance = 32) =>
  interpolate(frame, [start, start + 18], [distance, 0], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
  });

const Background: React.FC = () => {
  const frame = useCurrentFrame();
  const drift = frame * 0.6;

  return (
    <AbsoluteFill
      style={{
        backgroundColor: palette.bg,
        overflow: 'hidden',
      }}
    >
      <AbsoluteFill
        style={{
          backgroundImage: `
            radial-gradient(circle at 18% 22%, rgba(121, 242, 255, 0.2), transparent 34%),
            radial-gradient(circle at 82% 18%, rgba(255, 180, 83, 0.18), transparent 30%),
            radial-gradient(circle at 50% 78%, rgba(183, 255, 131, 0.12), transparent 28%),
            linear-gradient(180deg, ${palette.bgAlt} 0%, ${palette.bg} 100%)
          `,
        }}
      />
      <AbsoluteFill
        style={{
          opacity: 0.28,
          backgroundImage: `
            linear-gradient(rgba(149, 168, 199, 0.08) 1px, transparent 1px),
            linear-gradient(90deg, rgba(149, 168, 199, 0.08) 1px, transparent 1px)
          `,
          backgroundSize: '80px 80px',
          transform: `translateY(${(drift % 80) * -1}px)`,
        }}
      />
    </AbsoluteFill>
  );
};

const Chip: React.FC<{label: string; accent: string; soft: string}> = ({label, accent, soft}) => (
  <div
    style={{
      padding: '10px 18px',
      borderRadius: 999,
      border: `1px solid ${accent}55`,
      background: soft,
      color: accent,
      fontFamily: '"SF Mono", "Menlo", "Monaco", monospace',
      fontSize: 24,
      letterSpacing: 0.5,
    }}
  >
    {label}
  </div>
);

const StatCard: React.FC<{value: string; label: string; accent: string}> = ({value, label, accent}) => (
  <div
    style={{
      ...surfaceStyle,
      borderRadius: 32,
      padding: '34px 36px',
      display: 'flex',
      flexDirection: 'column',
      gap: 14,
      minWidth: 420,
    }}
  >
    <div
      style={{
        fontSize: 72,
        fontWeight: 700,
        letterSpacing: -2,
        color: accent,
      }}
    >
      {value}
    </div>
    <div
      style={{
        fontSize: 30,
        lineHeight: 1.3,
        color: palette.textMuted,
      }}
    >
      {label}
    </div>
  </div>
);

const SceneFrame: React.FC<{start: number; end: number; children: React.ReactNode}> = ({
  start,
  end,
  children,
}) => {
  const frame = useCurrentFrame();

  return (
    <AbsoluteFill
      style={{
        opacity: fadeRange(frame, start, end),
      }}
    >
      {children}
    </AbsoluteFill>
  );
};

const IntroScene: React.FC<{start: number}> = ({start}) => {
  const frame = useCurrentFrame();
  const titleSpring = spring({
    fps: 30,
    frame: frame - start,
    config: {
      damping: 14,
      stiffness: 110,
      mass: 0.8,
    },
  });
  const subOpacity = interpolate(frame, [start + 24, start + 52], [0, 1], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
  });

  return (
    <AbsoluteFill style={sceneStyle}>
      <div style={{display: 'flex', flexDirection: 'column', gap: 32}}>
        <div style={{display: 'flex', gap: 14}}>
          <Chip label="markdown" accent={palette.cyan} soft={palette.cyanSoft} />
          <Chip label="html" accent={palette.amber} soft={palette.amberSoft} />
          <Chip label="pdf" accent={palette.lime} soft="rgba(32, 52, 23, 0.8)" />
        </div>
        <div
          style={{
            maxWidth: 1240,
            fontSize: 102,
            lineHeight: 1.02,
            letterSpacing: -4,
            fontWeight: 700,
            transform: `scale(${interpolate(titleSpring, [0, 1], [0.92, 1])}) translateY(${interpolate(
              titleSpring,
              [0, 1],
              [36, 0],
            )}px)`,
            transformOrigin: 'left center',
          }}
        >
          Stop dumping whole documents into your context window.
        </div>
        <div
          style={{
            maxWidth: 1100,
            fontSize: 36,
            lineHeight: 1.45,
            color: palette.textMuted,
            opacity: subOpacity,
          }}
        >
          mq turns structure into a working index so the agent reads only what it needs.
        </div>
      </div>
    </AbsoluteFill>
  );
};

const ProblemScene: React.FC<{start: number}> = ({start}) => {
  const frame = useCurrentFrame();
  const panelSlide = rise(frame, start + 10, 42);
  const meter = interpolate(frame, [start + 12, start + 78], [0.18, 1], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
  });

  return (
    <AbsoluteFill style={{...sceneStyle, justifyContent: 'space-between'}}>
      <div
        style={{
          fontSize: 74,
          lineHeight: 1.06,
          letterSpacing: -2.8,
          fontWeight: 650,
          maxWidth: 1160,
        }}
      >
        AI agents waste tokens reading entire files.
      </div>
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '1.2fr 0.8fr',
          gap: 28,
          alignItems: 'stretch',
          transform: `translateY(${panelSlide}px)`,
        }}
      >
        <div style={terminalStyle}>
          <div style={{display: 'flex', gap: 10, marginBottom: 20}}>
            <div style={{width: 12, height: 12, borderRadius: 999, background: '#ff6b6b'}} />
            <div style={{width: 12, height: 12, borderRadius: 999, background: '#ffd166'}} />
            <div style={{width: 12, height: 12, borderRadius: 999, background: '#06d6a0'}} />
          </div>
          <div style={{fontSize: 30, color: palette.cyan}}>$ cat README.md</div>
          <div
            style={{
              marginTop: 20,
              fontSize: 23,
              lineHeight: 1.6,
              color: '#c4d4ee',
              opacity: 0.82,
              whiteSpace: 'pre-wrap',
            }}
          >
            {`# API Reference
Install steps...
Usage examples...
Configuration...
Troubleshooting...
Examples...
Release notes...
More headings...
More paragraphs...
More code blocks...`}
          </div>
        </div>
        <div
          style={{
            ...surfaceStyle,
            borderRadius: 32,
            padding: '32px 34px',
            display: 'flex',
            flexDirection: 'column',
            justifyContent: 'space-between',
          }}
        >
          <div>
            <div
              style={{
                fontSize: 26,
                color: palette.textMuted,
                marginBottom: 16,
              }}
            >
              Context pressure
            </div>
            <div
              style={{
                height: 22,
                borderRadius: 999,
                background: 'rgba(255,255,255,0.08)',
                overflow: 'hidden',
              }}
            >
              <div
                style={{
                  width: `${meter * 100}%`,
                  height: '100%',
                  background: 'linear-gradient(90deg, #ffb453 0%, #ff6b6b 100%)',
                }}
              />
            </div>
          </div>
          <div style={{display: 'flex', flexDirection: 'column', gap: 18}}>
            <div style={{fontSize: 58, fontWeight: 700, letterSpacing: -2}}>
              Full file in.
            </div>
            <div style={{fontSize: 58, fontWeight: 700, letterSpacing: -2, color: palette.amber}}>
              Signal buried.
            </div>
            <div style={{fontSize: 28, lineHeight: 1.4, color: palette.textMuted}}>
              The model spends budget on navigation before it can reason.
            </div>
          </div>
        </div>
      </div>
    </AbsoluteFill>
  );
};

const WorkflowScene: React.FC<{start: number}> = ({start}) => {
  const frame = useCurrentFrame();
  const steps = [
    {
      kicker: 'Step 1',
      title: '.tree',
      command: 'mq README.md .tree',
      detail: 'See the map before reading the terrain.',
      accent: palette.cyan,
    },
    {
      kicker: 'Step 2',
      title: `.search('authentication')`,
      command: `mq docs/ ".search('authentication')"`,
      detail: 'Narrow to the exact section or file.',
      accent: palette.amber,
    },
    {
      kicker: 'Step 3',
      title: `.section('API') | .text`,
      command: `mq doc.md ".section('API') | .text"`,
      detail: 'Extract only the content you need.',
      accent: palette.lime,
    },
  ];

  return (
    <AbsoluteFill style={sceneStyle}>
      <div style={{fontSize: 72, letterSpacing: -2.6, fontWeight: 650, marginBottom: 24}}>
        One query shape. Every format.
      </div>
      <div
        style={{
          fontSize: 30,
          lineHeight: 1.45,
          color: palette.textMuted,
          maxWidth: 1040,
          marginBottom: 42,
        }}
      >
        The same three-step pattern works everywhere: structure, search, then extract.
      </div>
      <div style={{display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 24}}>
        {steps.map((step, index) => {
          const delay = start + index * 18;
          const opacity = interpolate(frame, [delay, delay + 18], [0, 1], {
            extrapolateLeft: 'clamp',
            extrapolateRight: 'clamp',
          });
          const translate = rise(frame, delay, 42);

          return (
            <div
              key={step.title}
              style={{
                ...surfaceStyle,
                borderRadius: 32,
                padding: '28px 30px',
                minHeight: 420,
                opacity,
                transform: `translateY(${translate}px)`,
              }}
            >
              <div
                style={{
                  fontFamily: '"SF Mono", "Menlo", "Monaco", monospace',
                  fontSize: 22,
                  color: step.accent,
                  marginBottom: 18,
                }}
              >
                {step.kicker}
              </div>
              <div style={{fontSize: 52, letterSpacing: -2, fontWeight: 650, marginBottom: 22}}>
                {step.title}
              </div>
              <div
                style={{
                  ...terminalStyle,
                  padding: '18px 20px',
                  fontSize: 22,
                  color: '#dce7fa',
                }}
              >
                {step.command}
              </div>
              <div
                style={{
                  marginTop: 28,
                  fontSize: 28,
                  lineHeight: 1.4,
                  color: palette.textMuted,
                }}
              >
                {step.detail}
              </div>
            </div>
          );
        })}
      </div>
    </AbsoluteFill>
  );
};

const ResultsScene: React.FC<{start: number}> = ({start}) => {
  const frame = useCurrentFrame();
  const headlineOpacity = interpolate(frame, [start, start + 20], [0, 1], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
  });

  return (
    <AbsoluteFill style={sceneStyle}>
      <div style={{opacity: headlineOpacity}}>
        <div style={{fontSize: 74, letterSpacing: -2.4, fontWeight: 650, marginBottom: 18}}>
          Less waste. More reasoning.
        </div>
        <div
          style={{
            fontSize: 30,
            lineHeight: 1.45,
            color: palette.textMuted,
            maxWidth: 1200,
            marginBottom: 42,
          }}
        >
          These are the README numbers the video is built around, not invented demo metrics.
        </div>
      </div>
      <div style={{display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 26}}>
        {[
          {value: '83%', label: 'fewer markdown tokens when scoped correctly', accent: palette.cyan},
          {value: '50x', label: 'more PDFs searchable in a 200k-token window', accent: palette.amber},
          {value: '3-22ms', label: 'query latency on the markdown comparison benchmark', accent: palette.lime},
        ].map((stat, index) => {
          const delay = start + 10 + index * 14;
          const opacity = interpolate(frame, [delay, delay + 18], [0, 1], {
            extrapolateLeft: 'clamp',
            extrapolateRight: 'clamp',
          });
          const translate = rise(frame, delay, 34);

          return (
            <div key={stat.value} style={{opacity, transform: `translateY(${translate}px)`}}>
              <StatCard {...stat} />
            </div>
          );
        })}
      </div>
    </AbsoluteFill>
  );
};

const CtaScene: React.FC<{start: number}> = ({start}) => {
  const frame = useCurrentFrame();
  const commandOpacity = interpolate(frame, [start + 10, start + 30], [0, 1], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
  });
  const footerOpacity = interpolate(frame, [start + 38, start + 64], [0, 1], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
  });

  return (
    <AbsoluteFill style={sceneStyle}>
      <div
        style={{
          fontSize: 98,
          lineHeight: 1.02,
          letterSpacing: -4,
          fontWeight: 700,
          maxWidth: 1280,
        }}
      >
        Agentic querying for structured documents.
      </div>
      <div
        style={{
          marginTop: 28,
          fontSize: 36,
          lineHeight: 1.45,
          color: palette.textMuted,
          maxWidth: 1200,
        }}
      >
        One query model across Markdown, HTML, PDF, JSON, JSONL, and YAML.
      </div>
      <div
        style={{
          ...terminalStyle,
          marginTop: 42,
          width: 1080,
          opacity: commandOpacity,
        }}
      >
        <div style={{fontSize: 24, color: palette.textMuted, marginBottom: 18}}>Install</div>
        <div style={{fontSize: 28, color: palette.cyan}}>curl -fsSL https://raw.githubusercontent.com/muqsitnawaz/mq/main/install.sh | bash</div>
        <div style={{fontSize: 24, color: palette.textMuted, marginTop: 18, marginBottom: 14}}>Agent skill</div>
        <div style={{fontSize: 28, color: palette.amber}}>npx skills add muqsitnawaz/mq</div>
      </div>
      <div
        style={{
          marginTop: 28,
          fontSize: 24,
          letterSpacing: 0.3,
          color: palette.textMuted,
          opacity: footerOpacity,
        }}
      >
        mq &middot; structure first &middot; extract only what matters
      </div>
    </AbsoluteFill>
  );
};

export const Root: React.FC = () => {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const pulse = interpolate(Math.sin((frame / fps) * Math.PI * 0.75), [-1, 1], [0.85, 1]);

  const scenes = {
    intro: {start: 0, end: 120},
    problem: {start: 105, end: 225},
    workflow: {start: 210, end: 390},
    results: {start: 375, end: 510},
    cta: {start: 495, end: 630},
  };

  return (
    <>
      <Background />
      <AbsoluteFill
        style={{
          transform: `scale(${pulse})`,
          transformOrigin: 'center center',
        }}
      >
        {frame < scenes.intro.end && (
          <SceneFrame start={scenes.intro.start} end={scenes.intro.end}>
            <IntroScene start={scenes.intro.start} />
          </SceneFrame>
        )}
        {frame >= scenes.problem.start && frame < scenes.problem.end && (
          <SceneFrame start={scenes.problem.start} end={scenes.problem.end}>
            <ProblemScene start={scenes.problem.start} />
          </SceneFrame>
        )}
        {frame >= scenes.workflow.start && frame < scenes.workflow.end && (
          <SceneFrame start={scenes.workflow.start} end={scenes.workflow.end}>
            <WorkflowScene start={scenes.workflow.start} />
          </SceneFrame>
        )}
        {frame >= scenes.results.start && frame < scenes.results.end && (
          <SceneFrame start={scenes.results.start} end={scenes.results.end}>
            <ResultsScene start={scenes.results.start} />
          </SceneFrame>
        )}
        {frame >= scenes.cta.start && (
          <SceneFrame start={scenes.cta.start} end={scenes.cta.end}>
            <CtaScene start={scenes.cta.start} />
          </SceneFrame>
        )}
      </AbsoluteFill>
    </>
  );
};
