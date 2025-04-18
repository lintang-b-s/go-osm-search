# osm-search

Openstreetmap Full Text Search Engine (support Autocomplete & Spell Corrector Up to 2 Edit Distance per-word), Reverse Geocoder, Nearby Places without any external API/external database. Search Engine by default uses BM25F as the ranking function. The nearest neighbours query uses the R-tree data structure.

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
curl 'http://localhost:6060/api/search?query=Dunia gantadi&top_k=10&offset=0&lat=-6.17473908506388&lon=106.82749962074273'
```

### Autocomplete

```
curl 'http://localhost:6060/api/autocomplete?query=Kebun Binatang Ra&top_k=10&offset=0&lat=-6.17473908506388&lon=106.82749962074273'
```

### Reverse Geocoding

```
curl --location 'http://localhost:6060/api/reverse?lat=-6.179842&lon=106.749864'
```

### Nearby places With a Specific Openstreetmap Tag and Within a Specific Radius

```
curl --location 'http://localhost:6060/api/places?lat=-6.179842&lon=106.749864&feature=amenity=restaurant&k=10&offset=0&radius=3'
```

### Geofencing (In-memory)

```
curl --location 'http://localhost:6060/api/geofence' \
--header 'Content-Type: application/json' \
--data '{
    "fence_name": "ojol"
}'


curl --location --request PUT 'http://localhost:6060/api/geofence/ojol' \
--header 'Content-Type: application/json' \
--data '{
    "lat": -6.175263997609506,
    "lon": 106.82716214527025,
    "fence_point_name": "monumen_nasional",
    "radius": 1.2
}'



curl --location --request PUT 'http://localhost:6060/api/geofence/ojol/point' \
--header 'Content-Type: application/json' \
--data '{
    "lat":-6.169884724072774,
    "lon":106.8702583208934,
    "query_point_id": "ojol_budi"
}'

curl --location 'http://localhost:6060/api/geofence/ojol?lat=-6.17749341514094&lon=106.82291254922845&query_point_id=ojol_budi'

```
