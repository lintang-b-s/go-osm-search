package datastructure

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContains(t *testing.T) {
	c := NewCircle("a", -7.5680354571554025, 110.81169121664644, 1)

	t.Run("contains", func(t *testing.T) {

		if !c.Contains(-7.568015281898911, 110.81444088141711) {
			t.Error("Expected true, got false")
		}

		if !c.Contains(-7.572317914672147, 110.81118863253744) {
			t.Error("Expected true, got false")
		}

	})

	t.Run("not contains", func(t *testing.T) {

		if c.Contains(-7.559435821190102, 110.80760986341456) {
			t.Error("Expected false, got true")
		}

		if c.Contains(-7.55888752969384, 110.81268429828974) {
			t.Error("Expected false, got true")
		}
	})
}

func TestIsLineIntersectCircle(t *testing.T) {
	c := NewCircle("a", -7.559940429364888, 110.78890921003895, 1)

	t.Run("intersect", func(t *testing.T) {
		aLat, aLon := -7.5577436088673435, 110.78127272655398
		bLat, bLon := -7.564498664733181, 110.8035880873389

		assert.GreaterOrEqual(t, HaversineDistance(aLat, aLon, bLat, bLon), 2*c.radius)

		if !c.IsLineCircleIntersect(aLat, aLon, bLat, bLon) {
			t.Error("Expected intersect, got false")
		}

		aLat, aLon = -7.554174552910251, 110.76387434819563
		assert.GreaterOrEqual(t, HaversineDistance(aLat, aLon, bLat, bLon), 2*c.radius)

		if !c.IsLineCircleIntersect(aLat, aLon, bLat, bLon) {
			t.Error("Expected intersect, got false")
		}

	})

	t.Run("not intersect", func(t *testing.T) {
		aLat, aLon := -7.54644310927346, 110.77781694597039
		bLat, bLon := -7.552212073890144, 110.79527493164542

		if c.IsLineCircleIntersect(aLat, aLon, bLat, bLon) {
			t.Error("Expected not intersect, got intersect")
		}

		aLat, aLon = -7.556780022123904, 110.80638234773562
		bLat, bLon = -7.571822225335152, 110.80379309018404

		if c.IsLineCircleIntersect(aLat, aLon, bLat, bLon) {
			t.Error("Expected not intersect, got intersect")
		}

		aLat, aLon = -7.54419186409313, 110.77112430380268
		bLat, bLon = -7.561352173450039, 110.7642426738976

		if c.IsLineCircleIntersect(aLat, aLon, bLat, bLon) {
			t.Error("Expected not intersect, got intersect")
		}
	})

}
