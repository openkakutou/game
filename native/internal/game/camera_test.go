package game

import (
	"math"
	"testing"

	"github.com/openkakutou/stage"
)

// trainingStage carries the values of Training.def (quick-versus-bootstrap):
// localcoord 1280,720, boundleft/boundright -1420/1420, zoomout 0.5.
func trainingStage() *stage.Stage {
	return &stage.Stage{
		BGdef:        stage.BGdef{LocalCoordWidth: 1280, LocalCoordHeight: 720, ZoomOut: 0.5, ZoomIn: 0.8},
		CameraBounds: stage.CameraBounds{Left: -1420, Right: 1420},
	}
}

// trainingTension is Training.def's [Camera] tension.
const trainingTension = 400

func near(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestStageScale_LocalcoordWidthMapsOntoTheView(t *testing.T) {
	if s := StageScale(trainingStage(), 320); !near(s, 0.25) {
		t.Fatalf("scale = %v, want 0.25 (320/1280)", s)
	}
	if s := StageScale(&stage.Stage{}, 320); !near(s, 1) {
		t.Fatalf("scale without localcoord = %v, want 1 (MUGEN default 320)", s)
	}
}

func TestStageGroundShift_ShorterStageIsBottomAligned(t *testing.T) {
	// 720*0.25 = 180 rows of stage for a 240-row view: 60 rows below the top.
	if d := StageGroundShift(trainingStage(), 0.25, 240); !near(d, 60) {
		t.Fatalf("shift = %v, want 60", d)
	}
}

func TestLayerOrigin_SpriteAxisSitsOnTheStartPoint(t *testing.T) {
	// [BG BG]: start 0,0, delta 0,0, sprite 2,0 axis 364,361. Its left edge
	// lies left of the screen centre, not on it (the black left half).
	bg := stage.BGElement{StartX: 0, StartY: 0}
	x, y := LayerOrigin(bg, 364, 361, 0, 0.25, 320, 60)
	if !near(x, 160-91) || !near(y, 60-90.25) {
		t.Fatalf("origin = %v,%v, want 69,-30.25", x, y)
	}
}

func TestLayerOrigin_ScrollsAgainstTheCameraByDelta(t *testing.T) {
	// [BG Back]: start 0,412, delta 0.307; camera 100 game px = 400 stage units.
	back := stage.BGElement{StartY: 412, DeltaX: 0.307, DeltaY: 1}
	x, y := LayerOrigin(back, 364, 361, 100, 0.25, 320, 60)
	if !near(x, 160+0.25*(-400*0.307-364)) || !near(y, 60+0.25*(412-361)) {
		t.Fatalf("origin = %v,%v", x, y)
	}
}

func TestTileRun_InfiniteTileCoversTheView(t *testing.T) {
	first, n := TileRun(69, 182.25, 320, 1)
	if !near(first, 69-182.25) || n != 3 {
		t.Fatalf("run = %v x%d, want -113.25 x3", first, n)
	}
}

func TestTileRun_NoTileIsOneCopyAndCountedTileRepeats(t *testing.T) {
	if first, n := TileRun(69, 182.25, 320, 0); first != 69 || n != 1 {
		t.Fatalf("no tile = %v x%d", first, n)
	}
	if first, n := TileRun(69, 182.25, 320, 2); first != 69 || n != 2 {
		t.Fatalf("tile 2 = %v x%d", first, n)
	}
}

func TestStageCamera_StartsAtStartXAndHoldsWhenFightersStayCentred(t *testing.T) {
	c := NewStageCamera(trainingStage(), trainingTension, 0.25, 320)
	if c.Start() != 0 {
		t.Fatalf("start = %v", c.Start())
	}
	// p1startx/p2startx -280/280 stage units = -70/70 game px.
	if x := c.Follow(0, -70, 70, true); !near(x, 0) {
		t.Fatalf("x = %v, want 0", x)
	}
}

func TestStageCamera_SnapKeepsTheLeadingFighterATensionFromTheEdge(t *testing.T) {
	c := NewStageCamera(trainingStage(), trainingTension, 0.25, 320)
	// tension 400*0.25 = 100: rightest 240 pulls the right edge to 340.
	if x := c.Follow(0, 100, 240, true); !near(x, 170) {
		t.Fatalf("x = %v, want 170", x)
	}
}

func TestStageCamera_SmoothedStepEasesTowardTheTarget(t *testing.T) {
	c := NewStageCamera(trainingStage(), trainingTension, 0.25, 320)
	// Three easing passes of camera.go's tension smoothing, by hand.
	if x := c.Follow(0, 100, 240, false); !near(x, (-136.89475+185.95775)/2) {
		t.Fatalf("x = %v, want 24.5315", x)
	}
}

func TestStageCamera_StopsAtTheStageBound(t *testing.T) {
	c := NewStageCamera(trainingStage(), trainingTension, 0.25, 320)
	// boundright 1420*0.25 = 355, widened by halfWidth/zoomout - halfWidth = 160.
	if x := c.Follow(0, 600, 700, true); !near(x, 515) {
		t.Fatalf("x = %v, want 515", x)
	}
	if x := c.Follow(0, -700, -600, true); !near(x, -515) {
		t.Fatalf("x = %v, want -515", x)
	}
}
