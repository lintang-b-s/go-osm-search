# osm-search

## Benchmark

### Fulltext Search Query

|           BenchmarkFullTextQuery-12            | Iterations | Total ns/op  |  Total B/op | Total Allocs/op |
| :--------------------------------------------: | ---------- | :----------: | ----------: | --------------- |
|           BenchmarkFullTextQuery-12            | 2144       | 550372 ns/op | 596897 B/op | 2291            |
| BenchmarkFullTextQueryWithoutSpellCorrector-12 | 2521       | 468307 ns/op | 514499 B/op | 1428 allocs/op  |

Very slow hahah
