# mq Benchmark Results

Comparing agent performance with and without mq tool.

## Results

| Question | Mode | Input Tokens | Output Tokens | Wall Time (s) |
|----------|------|--------------|---------------|---------------|
| q1 | without_mq | 192667 | 107 | 24 |
| q1 | with_mq | 140822 | 299 | 28 |
| q2 | without_mq | 383244 | 326 | 39 |
| q2 | with_mq | 190591 | 282 | 30 |
| q3 | without_mq | 236929 | 39 | 25 |
| q3 | with_mq | 146334 | 294 | 33 |
| q4 | without_mq | 337225 | 24 | 31 |
| q4 | with_mq | 516897 | 931 | 85 |
| q5 | without_mq | 187768 | 37 | 21 |
| q5 | with_mq | 92334 | 253 | 21 |

## Summary

- **Questions tested**: 5
- **Total input tokens (without mq)**: 1337833
- **Total input tokens (with mq)**: 1086978
- **Input token reduction**: 20.0%
- **Total output tokens (without mq)**: 533
- **Total output tokens (with mq)**: 2059
- **Output token reduction**: -280.0%

## Interpretation

Input tokens represent the context consumed by the agent (files read, tool outputs).
A reduction in input tokens means mq helped the agent be more efficient by reading
only the relevant sections instead of entire files.
