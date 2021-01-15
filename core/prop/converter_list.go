package prop

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/otrego/clamshell/core/color"
	"github.com/otrego/clamshell/core/move"
	"github.com/otrego/clamshell/core/movetree"
)

// converters contain all the property converters.
var converters = []*SGFConverter{
	// Board Size
	&SGFConverter{
		Props: []Prop{"SZ"},
		Scope: RootScope,
		From: func(n *movetree.Node, prop string, data []string) error {
			if l := len(data); l != 1 {
				return fmt.Errorf("for prop %s, data  must be exactly 1, was %d", prop, l)
			}
			sz, err := strconv.Atoi(data[0])
			if err != nil {
				return fmt.Errorf("for prop %s, error parsing data %v as integer: %v", prop, data, err)
			}
			if sz < 1 || sz > 25 {
				return fmt.Errorf("for prop %s, size was %d, but must be between 1 and 25", prop, sz)
			}
			if n.GameInfo == nil {
				// For safety, make sure to set create gameinfo if it doesn't exist.
				n.GameInfo = &movetree.GameInfo{}
			}
			n.GameInfo.Size = sz
			return nil
		},
		To: func(n *movetree.Node) (string, error) {
			if n.GameInfo == nil {
				return "", nil
			}
			sz := n.GameInfo.Size
			if sz == 0 {
				// BoardSize is unspecified.
				return "", nil
			}
			if sz < 1 || sz > 25 {
				return "", fmt.Errorf("invalid board size: %d, but only values between 1 and 25 are allowed", sz)
			}
			return "SZ[" + strconv.Itoa(sz) + "]", nil
		},
	},

	// Placements
	&SGFConverter{
		Props: []Prop{"AB", "AW"},
		Scope: AllScope,
		From: func(n *movetree.Node, prop string, data []string) error {
			col, err := color.FromSGFProp(prop)
			if err != nil {
				return err
			}
			moves, err := move.ListFromSGFPoints(col, data)
			if err != nil {
				return err
			}
			n.Placements = append(n.Placements, moves...)
			return nil
		},
		To: func(n *movetree.Node) (string, error) {
			if len(n.Placements) == 0 {
				return "", nil
			}
			var black []string
			var white []string
			for _, mv := range n.Placements {
				sgfPt, err := mv.Point().ToSGF()
				if err != nil {
					return "", err
				}
				if mv.Color() == color.Black {
					black = append(black, sgfPt)
				} else if mv.Color() == color.White {
					white = append(white, sgfPt)
				}
			}
			var out strings.Builder
			if len(black) > 0 {
				out.WriteString("AB")
			}
			for _, pt := range black {
				out.WriteString("[" + pt + "]")
			}
			if len(white) > 0 {
				out.WriteString("AW")
			}
			for _, pt := range white {
				out.WriteString("[" + pt + "]")
			}
			return out.String(), nil
		},
	},

	// Moves
	&SGFConverter{
		Props: []Prop{"B", "W"},
		Scope: AllScope,
		From: func(n *movetree.Node, prop string, data []string) error {
			col, err := color.FromSGFProp(prop)
			if err != nil {
				return err
			}
			if n.Move != nil {
				return fmt.Errorf("found two moves on one node at move")
			}
			if len(data) != 1 && len(data) != 0 {
				return fmt.Errorf("expected black move data to have exactly one value or zero values")
			}
			if len(data) == 0 {
				data = []string{""}
			}
			move, err := move.FromSGFPoint(col, data[0])
			if err != nil {
				return err
			}
			n.Move = move
			return nil
		},
		To: func(n *movetree.Node) (string, error) {
			mv := n.Move
			if mv == nil {
				return "", nil
			}
			var col string
			if mv.Color() == color.Black {
				col = "B"
			} else if mv.Color() == color.White {
				col = "W"
			}
			if mv.IsPass() {
				// Return non-nil slice to indicate it should be stored.
				return col + "[]", nil
			}
			sgfPt, err := mv.Point().ToSGF()
			if err != nil {
				return "", err
			}
			return col + "[" + sgfPt + "]", nil
		},
	},

	// Komi
	&SGFConverter{
		Props: []Prop{"KM"},
		Scope: RootScope,
		From: func(n *movetree.Node, prop string, data []string) error {
			komi, err := strconv.ParseFloat(data[0], 64)
			if err != nil {
				return err
			}
			_, fp := math.Modf(komi)
			if !(fp == 0.5 || fp == 0.0) {
				return fmt.Errorf("for prop KM, value was %f, but the only decimal-value allowed for komi is .0 or .5", komi)
			}
			if n.GameInfo == nil {
				// For safety, make sure to set create gameinfo if it doesn't exist.
				n.GameInfo = &movetree.GameInfo{}
			}
			n.GameInfo.Komi = new(float64)
			*n.GameInfo.Komi = komi
			return nil
		},
		To: func(n *movetree.Node) (string, error) {
			if n.GameInfo == nil {
				return "", nil
			}
			if n.GameInfo.Komi == nil {
				return "", nil
			}
			komi := *n.GameInfo.Komi
			_, fp := math.Modf(komi)
			if !(fp == 0.5 || fp == 0.0) {
				return "", fmt.Errorf("invalid komi: the only decimal-value allowed for komi is .0 or .5. komi was %f", komi)
			}
			s := strconv.FormatFloat(komi, 'f', 1, 64)
			return fmt.Sprintf("KM[%s]", s), nil
		},
	},

	// PL. Initial Player Turn
	&SGFConverter{
		Props: []Prop{"PL"},
		Scope: RootScope,
		From: func(n *movetree.Node, prop string, data []string) error {
			if len(data) != 1 {
				return fmt.Errorf("PL property requires exactly 1 Value, but had %d", len(data))
			}
			if n.GameInfo == nil {
				// For safety, make sure to set create gameinfo if it doesn't exist.
				n.GameInfo = &movetree.GameInfo{}
			}

			switch data[0] {
			case "B", "b":
				n.GameInfo.Player = color.Black
			case "W", "w":
				n.GameInfo.Player = color.White
			default:
				return fmt.Errorf("Prop PL has invalid value of %s", data[0])
			}

			return nil
		},
		To: func(n *movetree.Node) (string, error) {
			if n.GameInfo == nil {
				return "", nil
			}

			switch n.GameInfo.Player {
			case "":
				return "", nil
			case "W", "B":
				return fmt.Sprintf("PL[%s]", n.GameInfo.Player), nil
			}
			return "", fmt.Errorf("prop PL can only have value W or B, but was %s", n.GameInfo.Player)
		},
	},
}

var propToConv = func(conv []*SGFConverter) map[Prop]*SGFConverter {
	mp := make(map[Prop]*SGFConverter)
	for _, c := range conv {
		for _, p := range c.Props {
			mp[p] = c
		}
	}
	return mp
}(converters)
