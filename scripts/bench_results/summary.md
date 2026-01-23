# mq Benchmark Results

Comparing agent performance with and without mq tool.

## Results

| Question | Mode | Input Tokens | Output Tokens | Wall Time (s) |
|----------|------|--------------|---------------|---------------|
| q1 | without_mq | 147070 | 52 | 23 |
| q1 | with_mq | 166501 | 220 | 25 |
| q2 | without_mq | 412668 | 265 | 37 |
| q2 | with_mq | 108225 | 117 | 19 |
| q3 | without_mq | 244271 | 291 | 24 |
| q3 | with_mq | 168318 | 154 | 27 |
| q4 | without_mq | 407773 | 142 | 36 |
| q4 | with_mq | 545708 | 216 | 56 |
| q5 | without_mq | 141917 | 130 | 19 |
| q5 | with_mq | 108618 | 116 | 22 |

## Summary

- **Questions tested**: 5
- **Total input tokens (without mq)**: 1353699
- **Total input tokens (with mq)**: 1097370
- **Input token reduction**: 20.0%
- **Total output tokens (without mq)**: 880
- **Total output tokens (with mq)**: 823
- **Output token reduction**: 10.0%

## Interpretation

Input tokens represent the context consumed by the agent (files read, tool outputs).
A reduction in input tokens means mq helped the agent be more efficient by reading
only the relevant sections instead of entire files.
