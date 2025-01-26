# osm-search
Openstreetmap Full Text Search (support Autocomplete & Spell Corrector) without any external API/external database.

# Quick Start
## Indexing
```
1. download the jabodetabek openstreetmap pbf file at: https://drive.google.com/file/d/1MZfZhFAFLUGouAeQK8-g-S4O2HBDHLRn/view?usp=sharing
Note: or you can also use another openstreetmap file with the osm.pbf format (https://download.geofabrik.de/)
2. go mod tidy &&  mkdir bin
3. go build -o ./bin/osm-search-indexer ./cmd/indexing 
4. ./bin/osm-search-indexer -f "jabodetabek_big.osm.pbf"
Note: The indexing process takes 3-5 minutes, please wait. you can also replace the osm pbf file that you want to use.
5. run the server
```

## Server
```
1. go build -o ./bin/osm-search-server ./cmd/server 
2. ./bin/osm-search-server
```



## Benchmark

|          BenchmarkName          | Iterations | Total ns/op  |  Total B/op | Total Allocs/op |
| :-----------------------------: | ---------- | :----------: | ----------: | --------------- |
| BenchmarkFullTextSearchQuery-12 | 2930       | 360077 ns/op | 413571 B/op | 1516 allocs/op  |
|    BenchmarkAutocomplete-12     | 3816       | 288859 ns/op | 246140 B/op | 819 allocs/op   |

Very slow. but this project is much faster than nominatim when I compare the load test results using k6 on my laptop :).


