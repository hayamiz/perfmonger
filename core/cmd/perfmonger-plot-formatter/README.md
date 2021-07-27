# perfmonger-plot-formatter

`perfmonger-plot-formatter` is a command to generate data files for plotting
various metrices recorded by `perfmonger-recorder`.

## Notes on memory usage records

### `free(1)` compatible summarization

- total: `MemStat.MemTotal`
- used: `if (memUsed < 0) { MemStat.MemTotal - MemStat.MemFree } else { memUsed }`
  - memUsed := `MemStat.MemTotal - MemStat.MemFree - mainCached - MemStat.Buffers`
  - mainCached := `MemStat.Cached + MemStat.SReclaimable`
- free: `MemStat.MemFree`
- shared: `MemStat.Shmem`
- buffers: `MemStat.Buffers`
- cache: `mainCached`
  - mainCached := `MemStat.Cached + MemStat.SReclaimable`
- available: `MemStat.MemAvailable`

### Additional info

- hugeTotal: `MemStat.HugePages_Total`
- hugeUsed: `MemStat.HugePages_Total - MemStat.HugePages_Free`
