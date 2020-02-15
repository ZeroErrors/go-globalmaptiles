// Ported from Python implementation https://gist.github.com/maptiler/fddb5ce33ba995d5523de9afdf8ef118
package globalmaptiles

import (
	"log"
	"math"
	"strconv"
)

// https://www.maptiler.com/google-maps-coordinates-tile-bounds-projection/
// https://gist.github.com/maptiler/fddb5ce33ba995d5523de9afdf8ef118
type GlobalMercator struct {
	tileSize          int
	initialResolution float64
	originShift       float64
}

// Initialize the TMS Global Mercator pyramid
func NewGlobalMercator(tileSize int) *GlobalMercator {
	return &GlobalMercator{
		tileSize:          tileSize,
		initialResolution: 2 * math.Pi * 6378137 / float64(tileSize), // 156543.03392804062 for tileSize 256 pixels
		originShift:       2 * math.Pi * 6378137 / 2.0,               // 20037508.342789244
	}
}

func (m GlobalMercator) TileSize() int {
	return m.tileSize
}

// Converts given lat/lon in WGS84 Datum to XY in Spherical Mercator EPSG:900913
func (m GlobalMercator) LatLonToMeters(lat, lon float64) (mx, my float64) {
	mx = lon * m.originShift / 180.0
	my = math.Log(math.Tan((90+lat)*math.Pi/360.0)) / (math.Pi / 180.0)

	my = my * m.originShift / 180.0
	return mx, my
}

// Converts XY point from Spherical Mercator EPSG:900913 to lat/lon in WGS84 Datum
func (m GlobalMercator) MetersToLatLon(mx, my float64) (lat, lon float64) {
	lon = (mx / m.originShift) * 180.0
	lat = (my / m.originShift) * 180.0

	lat = 180 / math.Pi * (2*math.Atan(math.Exp(lat*math.Pi/180.0)) - math.Pi/2.0)
	return lat, lon
}

// Converts pixel coordinates in given zoom level of pyramid to EPSG:900913
func (m GlobalMercator) PixelsToMeters(px, py float64, zoom int) (mx, my float64) {
	res := m.Resolution(zoom)
	mx = px*res - m.originShift
	my = py*res - m.originShift
	return mx, my
}

// Converts EPSG:900913 to pyramid pixel coordinates in given zoom level
func (m GlobalMercator) MetersToPixels(mx, my float64, zoom int) (px, py float64) {
	res := m.Resolution(zoom)
	px = (mx + m.originShift) / res
	py = (my + m.originShift) / res
	return px, py
}

// Returns a tile covering region in given pixel coordinates
func (m GlobalMercator) PixelsToTile(px, py float64) (tx, ty int) {
	tx = int(math.Ceil(px/float64(m.tileSize)) - 1)
	ty = int(math.Ceil(py/float64(m.tileSize)) - 1)
	return tx, ty
}

// Move the origin of pixel coordinates to top-left corner
func (m GlobalMercator) PixelsToRaster(px, py float64, zoom int) (rpx, rpy float64) {
	mapSize := m.tileSize << uint(zoom)
	return px, float64(mapSize) - py
}

// Returns tile for given mercator coordinates
func (m GlobalMercator) MetersToTile(mx, my float64, zoom int) (tx, ty int) {
	px, py := m.MetersToPixels(mx, my, zoom)
	return m.PixelsToTile(px, py)
}

func (m GlobalMercator) TileToPixels(tx, ty int) (px, py float64) {
	return float64(tx * m.tileSize), float64(ty * m.tileSize)
}

func (m GlobalMercator) TileToMeters(tx, ty int, zoom int) (mx, my float64) {
	return m.PixelsToMeters(float64(tx*m.tileSize), float64(ty*m.tileSize), zoom)
}

// Returns bounds of the given tile in EPSG:900913 coordinates
func (m GlobalMercator) TileBounds(tx, ty int, zoom int) (minx, miny, maxx, maxy float64) {
	minx, miny = m.PixelsToMeters(float64(tx*m.tileSize), float64(ty*m.tileSize), zoom)
	maxx, maxy = m.PixelsToMeters(float64((tx+1)*m.tileSize), float64((ty+1)*m.tileSize), zoom)
	return minx, miny, maxx, maxy
}

// Returns bounds of the given tile in latutude/longitude using WGS84 datum
func (m GlobalMercator) TileLatLonBounds(tx, ty int, zoom int) (minLat, minLon, maxLat, maxLon float64) {
	minx, miny, maxx, maxy := m.TileBounds(tx, ty, zoom)
	minLat, minLon = m.MetersToLatLon(minx, miny)
	maxLat, maxLon = m.MetersToLatLon(maxx, maxy)
	return minLat, minLon, maxLat, maxLon
}

// Resolution (meters/pixel) for given zoom level (measured at Equator)
func (m GlobalMercator) Resolution(zoom int) float64 {
	// return (2 * math.Pi * 6378137) / (m.tileSize * 2**zoom)
	return m.initialResolution / math.Pow(2, float64(zoom))
}

// Maximal scaledown zoom of the pyramid closest to the pixelSize.
func (m GlobalMercator) ZoomForPixelSize(pixelSize float64) int {
	for i := 0; i < 30; i++ {
		if pixelSize > m.Resolution(i) {
			if i != 0 {
				return i - 1
			} else {
				return 0 // We don't want to scale up
			}
		}
	}
	log.Panicf("invalid pixel size: %v", pixelSize)
	return 0
}

// Converts TMS tile coordinates to Google Tile coordinates
func (m GlobalMercator) GoogleTile(tx, ty int, zoom int) (gtx, gty int) {
	// coordinate origin is moved from bottom-left to top-left corner of the extent
	return tx, int(math.Pow(2, float64(zoom))-1) - ty
}

// Converts TMS tile coordinates to Microsoft QuadTree
func (m GlobalMercator) QuadTree(tx, ty, zoom int) string {
	quadKey := ""
	ty = int(math.Pow(2, float64(zoom))-1) - ty
	for i := zoom; i > 0; i-- {
		digit := 0
		mask := 1 << uint(i-1)
		if (tx & mask) != 0 {
			digit += 1
		}
		if (ty & mask) != 0 {
			digit += 2
		}
		quadKey += strconv.Itoa(digit)
	}
	return quadKey
}

// TODO: Implement GlobalGeodetic
