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
http://localhost:6060/api/places?lat=-6.179842&lon=106.749864&feature=amenity=restaurant&k=10&offset=2&radius=3
```




