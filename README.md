# osm-search
Openstreetmap Full Text Search Engine (support Autocomplete & Spell Corrector Up to 2 Edit Distance per-word), Reverse Geocoder, Nearest Places without any external API/external database. Search Engine by default uses BM25F as the ranking function. The nearest neighbours query uses the R-tree data structure.

# Quick Start
## Indexing
```
1. download the jabodetabek openstreetmap pbf file at: https://drive.google.com/file/d/1MZfZhFAFLUGouAeQK8-g-S4O2HBDHLRn/view?usp=sharing
Note: or you can also use another openstreetmap file with the osm.pbf format (https://download.geofabrik.de/)
2. go mod tidy &&  mkdir bin
3. go build -o ./bin/osm-search-indexer ./cmd/indexing 
4. ./bin/osm-search-indexer -f "jabodetabek_big.osm.pbf"
Note: The indexing process takes 1-3 minutes, please wait. you can also replace the osm pbf file that you want to use.
5. run the server
```

## Server
```
1. go build -o ./bin/osm-search-server ./cmd/server 
2. ./bin/osm-search-server
```

## Feature

### Search With Spell Correction
```
curl --location --request GET 'http://localhost:6060/api/search' \
--header 'Content-Type: application/json' \
--data '{
    "query": "Dunia Pantadi",
    "top_k": 10,
    "offset": 0,
    "lat": -6.17473908506388,
    "lon":  106.82749962074273
}'
```

### Autocomplete
```
curl --location --request GET 'http://localhost:6060/api/autocomplete' \
--header 'Content-Type: application/json' \
--data '{
    "query": "Kebun Binatang Ra",
    "top_k": 10,
    "lat": -6.17473908506388,
    "lon":  106.82749962074273
    }'
```

### Reverse Geocoding
```
curl --location 'http://localhost:6060/api/reverse?lat=-6.179842&lon=106.749864'
```

### Nearest places With a Specific Openstreetmap Tag and Within a Specific Radius
```
curl --location 'http://localhost:6060/api/places?lat=-6.179842&lon=106.749864&feature=amenity=restaurant&k=10&offset=2&radius=3'
```





## Appendix
### Data Race Detection

```
 go build  -race  -o ./bin/osm-search-indexer ./cmd/indexing
 go build -race  -o ./bin/osm-search-server ./cmd/server
 go test -race ./cmd/indexing/
go test -race ./cmd/server/


```

### Goroutine Leak Detection
(using https://github.com/uber-go/goleak)
```
go test -race ./cmd/indexing/
go test -race ./cmd/server/
```

### Heap Escape Analysis

```
go build -gcflags "-m"  ./cmd/indexing/main.go
go build -gcflags "-m"  ./cmd/server/main.go
```

### pprof 

```
(pprof indexer): ./bin/osm-search-indexer -f "jabodetabek_big.osm.pbf" -cpuprofile=osmsearchcpu.prof -memprofile=osmsearchmem.mprof
NOTE: indexing process run slower if you use this.
(pprof server): ./bin/osm-server



cd pkg/[package_inside_pkg] && go test -bench . ./... -benchmem -cpuprofile prof.cpu -memprofile prof.mem
NOTE: for "searcher" package you must copy index directory "lintang"  & "docs_store.db" to searcher package


command (open pprof cpu/heap):
go tool pprof [binary] [file_name].cpu
go tool pprof [binary] [file_name].mprof


command (open pprof cpu/heap) example:
go  tool pprof ./bin/osm-search-indexer  osmsearchcpu.prof
go  tool pprof ./bin/osm-search-indexer  osmsearchmem.mprof

go tool pprof pkg.test prof.cpu



go tool pprof -alloc_objects pkg.test prof.mem

command pprof on web:

go tool pprof -http=":8081" [binary] [profile]

go tool pprof -http=":8081" ./bin/osm-search-indexer  osmsearchcpu.prof

go tool pprof -http=":8081" pkg.test prof.cpu

go tool pprof -http=":8082" pkg.test prof.mem


some useful command inside pprof:
top10
list [functionName]
web
web [functionName]
disasm [functionName]


pprof web flamegraph:
go tool pprof -http=":8081" [binary] [profile]
[Switching to the Flame graph view (via the View menu) will display a flame graph. This view provides a compact representation of caller/callee relations:]


pprof web go-osm-search-server:
load test first: cd docs/load_tests/ && k6 run search.js
curl  --output go-osm-search-server-profile    'http://localhost:6060/debug/pprof/profile?seconds=20'


go tool pprof -http=":8081" go-osm-search-server-profile

open view/flame graph

or 
go tool pprof  go-osm-search-server-profile
top10
list [functionName]
```

### Simple K6 Load Test
```

1. go build -o ./bin/osm-search-server ./cmd/server 
2. ./bin/osm-search-server
3. cd docs/load_tests/ && k6 run search.js
```
