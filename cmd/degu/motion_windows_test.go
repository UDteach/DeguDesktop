//go:build windows

package main

import "testing"

func TestHorizontalMotionFramesUseStableRightFacingSequence(t *testing.T) {
	allowed := map[int]bool{
		walkStart:     true,
		walkStart + 1: true,
		walkStart + 3: true,
	}
	states := []behaviorState{
		stateWalk,
		stateScurry,
		stateWheel,
		stateForage,
		stateCarry,
	}

	for _, state := range states {
		for frame := 0; frame < 64; frame++ {
			got := currentFrame(state, frame)
			if !allowed[got] {
				t.Fatalf("currentFrame(%v, %d) = %d, want stable right-facing walk frame", state, frame, got)
			}
		}
	}
}

func TestFrameFromSeqHandlesEmptyAndBadDivisor(t *testing.T) {
	if got := frameFromSeq(nil, 12, 2); got != idleStart {
		t.Fatalf("frameFromSeq(nil) = %d, want %d", got, idleStart)
	}
	seq := []int{7, 9}
	if got := frameFromSeq(seq, 3, 0); got != 9 {
		t.Fatalf("frameFromSeq with zero divisor = %d, want 9", got)
	}
}

func TestFrameFromSeqClampedHoldsFinalFrame(t *testing.T) {
	seq := []int{7, 9, 11}
	if got := frameFromSeqClamped(seq, 999, 2); got != 11 {
		t.Fatalf("frameFromSeqClamped past end = %d, want 11", got)
	}
	if got := frameFromSeqClamped(seq, 3, 0); got != 11 {
		t.Fatalf("frameFromSeqClamped with zero divisor = %d, want 11", got)
	}
}

func TestTypingStartsAndExtendsWheelOnlyInKeyboardMode(t *testing.T) {
	a := &petApp{
		mode:         modeKeyboard,
		wheelEnabled: true,
		wheelX:       400,
		sceneW:       1200,
		speed:        3,
		pets: []deguPet{
			{state: stateWalk, stateTicks: 12, item: noItem},
			{state: stateWalk, stateTicks: 12, item: noItem},
		},
	}

	a.onTyping()
	if got := a.pets[0].state; got != stateWheel {
		t.Fatalf("first pet state = %v, want stateWheel", got)
	}
	if got := a.pets[0].stateTicks; got != wheelKeyHold {
		t.Fatalf("wheel hold ticks = %d, want %d", got, wheelKeyHold)
	}
	if got := a.pets[0].moveSpeed; got != 0 {
		t.Fatalf("wheel pet moveSpeed = %d, want 0", got)
	}
	wantX := clamp(a.wheelX-wheelSize/2, 0, max(0, a.sceneW-spriteW))
	if got := a.pets[0].x; got != wantX {
		t.Fatalf("wheel pet x = %d, want %d", got, wantX)
	}
	if got := a.pets[1].state; got != stateScurry {
		t.Fatalf("second pet state = %v, want stateScurry", got)
	}

	a.pets[0].frame = 7
	a.pets[0].stateTicks = 3
	a.onTyping()
	if got := a.pets[0].frame; got != 7 {
		t.Fatalf("wheel frame reset while extending: got %d, want 7", got)
	}
	if got := a.pets[0].stateTicks; got != wheelKeyHold {
		t.Fatalf("extended wheel hold ticks = %d, want %d", got, wheelKeyHold)
	}
}

func TestTypingDoesNotStartWheelInRandomMode(t *testing.T) {
	a := &petApp{
		mode:         modeRandom,
		wheelEnabled: true,
		wheelX:       400,
		sceneW:       1200,
		pets: []deguPet{
			{state: stateWalk, stateTicks: 12, item: noItem},
		},
	}

	a.onTyping()
	if got := a.pets[0].state; got == stateWheel {
		t.Fatalf("typing in random mode started wheel state")
	}
}

func TestTurnStateUsesGeneratedTurnFrames(t *testing.T) {
	if got := currentFrame(stateTurn, 0); got != turnStart {
		t.Fatalf("turn frame 0 = %d, want %d", got, turnStart)
	}
	if got := currentFrame(stateTurn, turnTicks-1); got != turnStart+turnFrames-1 {
		t.Fatalf("turn final active frame = %d, want %d", got, turnStart+turnFrames-1)
	}
	if got := currentFrame(stateTurn, turnTicks+10); got != turnStart+turnFrames-1 {
		t.Fatalf("turn frame after duration = %d, want held final frame %d", got, turnStart+turnFrames-1)
	}
}

func TestTurnDrawDirectionMirrorsOnlyLeftToRightTurns(t *testing.T) {
	if got := turnDrawDirection(1, -1); got != 1 {
		t.Fatalf("right-to-left turn draw direction = %d, want 1", got)
	}
	if got := turnDrawDirection(-1, 1); got != -1 {
		t.Fatalf("left-to-right turn draw direction = %d, want -1", got)
	}
}

func TestSetBidirectionalOffNormalizesPets(t *testing.T) {
	a := &petApp{
		bidirectional: true,
		speed:         3,
		pets: []deguPet{
			{state: stateTurn, dir: -1, nextDir: -1, item: noItem},
			{state: stateWalk, dir: -1, nextDir: -1, item: noItem},
		},
	}

	a.setBidirectional(false)
	if a.bidirectional {
		t.Fatalf("bidirectional stayed enabled")
	}
	for i, pet := range a.pets {
		if pet.dir != 1 || pet.nextDir != 1 {
			t.Fatalf("pet %d direction = (%d,%d), want (1,1)", i, pet.dir, pet.nextDir)
		}
		if pet.state == stateTurn {
			t.Fatalf("pet %d remained in stateTurn", i)
		}
	}
}
