package geo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPointInsidePolygon(t *testing.T) {
	polygon := [][]float64{
		{-7.8236786093625454, 110.32093322132368},
		{-7.829740180582352, 110.35293804508764},
		{-7.826476268571158, 110.4094171458476},
		{-7.7821777971150485, 110.4098878050206},
		{-7.7821777971150485, 110.43012614945958},
		{-7.763058061783706, 110.43012614945958},
		{-7.742538353844481, 110.34211288410864},
	}

	plLat := make([]float64, len(polygon))
	plLon := make([]float64, len(polygon))

	for i, p := range polygon {
		plLat[i] = p[0]
		plLon[i] = p[1]
	}

	t.Run("point inside polygon", func(t *testing.T) {
		point := []float64{-7.786841015007818, 110.35482068177964}

		bb := NewBoundingBox(plLat, plLon)
		contain := bb.Contains(point[0], point[1])
		assert.Equal(t, true, contain)
	})

	t.Run("point outside polygon", func(t *testing.T) {
		point := []float64{-7.709038594647804, 110.5904486305967}

		bb := NewBoundingBox(plLat, plLon)
		contain := bb.Contains(point[0], point[1])
		assert.False(t, contain)
	})

	t.Run("point inside polygon", func(t *testing.T) {
		point := []float64{-7.786841015007818, 110.35482068177964}
		isInside := IsPointInsidePolygonWindingNum(point[0], point[1], plLat, plLon)
		assert.True(t, isInside)
	})

	t.Run("point outside polygon", func(t *testing.T) {
		point := []float64{-7.709038594647804, 110.5904486305967}
		isInside := IsPointInsidePolygonWindingNum(point[0], point[1], plLat, plLon)
		assert.False(t, isInside)
	})

	
}
