# osm-search
Openstreetmap Full Text Search (support Autocomplete & Spell Corrector Up to 2 Edit Distance per-word) and Reverse Geocoder without any external API/external database. by default uses BM25F as the ranking function.

# Quick Start
## Indexing
```
1. download the jabodetabek openstreetmap pbf file at: https://drive.google.com/file/d/1MZfZhFAFLUGouAeQK8-g-S4O2HBDHLRn/view?usp=sharing
Note: or you can also use another openstreetmap file with the osm.pbf format (https://download.geofabrik.de/)
2. go mod tidy &&  mkdir bin
3. go build -o ./bin/osm-search-indexer ./cmd/indexing 
4. ./bin/osm-search-indexer -f "jabodetabek_big.osm.pbf"
Note: The indexing process takes 3-7 minutes, please wait. you can also replace the osm pbf file that you want to use.
5. run the server
```

## Server
```
1. go build -o ./bin/osm-search-server ./cmd/server 
2. ./bin/osm-search-server
```






