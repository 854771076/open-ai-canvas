package app

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// 参数化基础组件的参数白名单，与 web/src/lib/canvas/previs/previs-prop-catalog.ts 的 params 一一对应。
type cloudAgentPrevisParamSpec struct {
	Key      string
	Min, Max float64
	Default  float64
	Integer  bool
	Options  []string // 非空表示枚举参数，第一个为默认值
}

func previsNum(key string, min, max, def float64, integer bool) cloudAgentPrevisParamSpec {
	return cloudAgentPrevisParamSpec{Key: key, Min: min, Max: max, Default: def, Integer: integer}
}

func previsEnum(key string, options ...string) cloudAgentPrevisParamSpec {
	return cloudAgentPrevisParamSpec{Key: key, Options: options}
}

var previsMaterials = []string{"plaster", "brick", "stone", "concrete", "wood"}

var cloudAgentPrevisPropParams = map[string][]cloudAgentPrevisParamSpec{
	"shotloom-shape-box":                 {previsNum("width", 0.02, 500, 1, false), previsNum("height", 0.02, 500, 1, false), previsNum("depth", 0.02, 500, 1, false)},
	"shotloom-shape-sphere":              {previsNum("width", 0.02, 500, 1, false), previsNum("height", 0.02, 500, 1, false), previsNum("depth", 0.02, 500, 1, false), previsNum("segments", 6, 96, 24, true), previsNum("crossSegments", 6, 96, 16, true)},
	"shotloom-shape-cylinder":            {previsNum("width", 0.02, 500, 1, false), previsNum("height", 0.02, 500, 1, false), previsNum("depth", 0.02, 500, 1, false), previsNum("segments", 6, 96, 24, true)},
	"shotloom-shape-cone":                {previsNum("width", 0.02, 500, 1, false), previsNum("height", 0.02, 500, 1, false), previsNum("depth", 0.02, 500, 1, false), previsNum("segments", 6, 96, 24, true)},
	"shotloom-shape-capsule":             {previsNum("width", 0.02, 500, 1, false), previsNum("height", 0.02, 500, 1, false), previsNum("depth", 0.02, 500, 1, false), previsNum("segments", 6, 96, 16, true), previsNum("crossSegments", 6, 96, 6, true)},
	"shotloom-shape-torus":               {previsNum("width", 0.02, 500, 1, false), previsNum("height", 0.02, 500, 1, false), previsNum("depth", 0.02, 500, 1, false), previsNum("segments", 6, 96, 32, true), previsNum("crossSegments", 6, 96, 12, true), previsNum("thickness", 0.03, 0.45, 0.18, false)},
	"shotloom-shape-pyramid":             {previsNum("width", 0.02, 500, 1, false), previsNum("height", 0.02, 500, 1, false), previsNum("depth", 0.02, 500, 1, false)},
	"shotloom-shape-plane":               {previsNum("width", 0.02, 500, 1, false), previsNum("height", 0.02, 500, 0.04, false), previsNum("depth", 0.02, 500, 1, false)},
	"shotloom-shape-wedge":               {previsNum("width", 0.02, 500, 1, false), previsNum("height", 0.02, 500, 1, false), previsNum("depth", 0.02, 500, 1, false)},
	"shotloom-shape-ramp":                {previsNum("width", 0.02, 500, 1, false), previsNum("height", 0.02, 500, 1, false), previsNum("depth", 0.02, 500, 1, false), previsNum("thickness", 0.03, 0.45, 0.18, false)},
	"shotloom-shape-arch":                {previsNum("width", 0.02, 500, 1, false), previsNum("height", 0.02, 500, 1, false), previsNum("depth", 0.02, 500, 1, false), previsNum("segments", 6, 96, 16, true), previsNum("thickness", 0.03, 0.45, 0.18, false)},
	"shotloom-shape-hemisphere":          {previsNum("width", 0.02, 500, 1, false), previsNum("height", 0.02, 500, 1, false), previsNum("depth", 0.02, 500, 1, false), previsNum("segments", 6, 96, 24, true), previsNum("crossSegments", 6, 96, 12, true)},
	"shotloom-shape-tube":                {previsNum("width", 0.02, 500, 1, false), previsNum("height", 0.02, 500, 1, false), previsNum("depth", 0.02, 500, 1, false), previsNum("segments", 6, 96, 24, true), previsNum("thickness", 0.03, 0.45, 0.18, false)},
	"shotloom-shape-l":                   {previsNum("width", 0.02, 500, 1, false), previsNum("height", 0.02, 500, 1, false), previsNum("depth", 0.02, 500, 1, false), previsNum("thickness", 0.03, 0.45, 0.18, false)},
	"shotloom-shape-u":                   {previsNum("width", 0.02, 500, 1, false), previsNum("height", 0.02, 500, 1, false), previsNum("depth", 0.02, 500, 1, false), previsNum("thickness", 0.03, 0.45, 0.18, false)},
	"shotloom-shape-arc":                 {previsNum("width", 0.02, 500, 1, false), previsNum("height", 0.02, 500, 1, false), previsNum("depth", 0.02, 500, 1, false), previsNum("segments", 6, 96, 32, true), previsNum("crossSegments", 6, 96, 12, true), previsNum("angle", 10, 360, 180, false), previsNum("thickness", 0.03, 0.45, 0.18, false)},
	"shotloom-furniture-chair":           {previsNum("width", 0.2, 20, 0.48, false), previsNum("height", 0.2, 20, 0.9, false), previsNum("depth", 0.2, 20, 0.5, false), previsNum("surfaceHeight", 0.2, 20, 0.45, false)},
	"shotloom-furniture-stool":           {previsNum("width", 0.2, 20, 0.38, false), previsNum("height", 0.2, 20, 0.44, false), previsNum("depth", 0.2, 20, 0.38, false)},
	"shotloom-furniture-bench":           {previsNum("width", 0.2, 20, 1.6, false), previsNum("height", 0.2, 20, 0.46, false), previsNum("depth", 0.2, 20, 0.43, false)},
	"shotloom-furniture-armchair":        {previsNum("width", 0.2, 20, 0.65, false), previsNum("height", 0.2, 20, 0.93, false), previsNum("depth", 0.2, 20, 0.62, false), previsNum("surfaceHeight", 0.2, 20, 0.45, false)},
	"shotloom-furniture-sofa-single":     {previsNum("width", 0.2, 20, 0.92, false), previsNum("height", 0.2, 20, 0.88, false), previsNum("depth", 0.2, 20, 0.85, false), previsNum("surfaceHeight", 0.2, 20, 0.43, false)},
	"shotloom-furniture-sofa":            {previsNum("width", 0.2, 20, 2.2, false), previsNum("height", 0.2, 20, 0.9, false), previsNum("depth", 0.2, 20, 0.9, false), previsNum("surfaceHeight", 0.2, 20, 0.43, false)},
	"shotloom-furniture-barstool":        {previsNum("width", 0.2, 20, 0.43, false), previsNum("height", 0.2, 20, 0.78, false), previsNum("depth", 0.2, 20, 0.43, false)},
	"shotloom-furniture-lounger":         {previsNum("width", 0.2, 20, 0.72, false), previsNum("height", 0.2, 20, 0.94, false), previsNum("depth", 0.2, 20, 1.8, false), previsNum("surfaceHeight", 0.2, 20, 0.36, false)},
	"shotloom-furniture-table":           {previsNum("width", 0.2, 20, 1.4, false), previsNum("height", 0.2, 20, 0.76, false), previsNum("depth", 0.2, 20, 0.85, false)},
	"shotloom-furniture-roundtable":      {previsNum("width", 0.2, 20, 1.2, false), previsNum("height", 0.2, 20, 0.75, false), previsNum("depth", 0.2, 20, 1.2, false)},
	"shotloom-furniture-desk":            {previsNum("width", 0.2, 20, 1.3, false), previsNum("height", 0.2, 20, 0.75, false), previsNum("depth", 0.2, 20, 0.65, false)},
	"shotloom-furniture-coffeetable":     {previsNum("width", 0.2, 20, 1.05, false), previsNum("height", 0.2, 20, 0.42, false), previsNum("depth", 0.2, 20, 0.6, false)},
	"shotloom-furniture-counter":         {previsNum("width", 0.2, 20, 2, false), previsNum("height", 0.2, 20, 1.08, false), previsNum("depth", 0.2, 20, 0.65, false)},
	"shotloom-furniture-wardrobe":        {previsNum("width", 0.2, 20, 1.5, false), previsNum("height", 0.2, 20, 2.1, false), previsNum("depth", 0.2, 20, 0.58, false), previsNum("opening", 0, 120, 0, false)},
	"shotloom-furniture-bookshelf":       {previsNum("width", 0.2, 20, 0.9, false), previsNum("height", 0.2, 20, 1.9, false), previsNum("depth", 0.2, 20, 0.32, false), previsNum("layers", 2, 16, 5, true)},
	"shotloom-furniture-rack":            {previsNum("width", 0.2, 20, 1.2, false), previsNum("height", 0.2, 20, 1.8, false), previsNum("depth", 0.2, 20, 0.55, false), previsNum("layers", 2, 16, 4, true)},
	"shotloom-furniture-bed-single":      {previsNum("width", 0.2, 20, 1, false), previsNum("height", 0.2, 20, 1, false), previsNum("depth", 0.2, 20, 2.05, false), previsNum("surfaceHeight", 0.2, 20, 0.5, false)},
	"shotloom-furniture-bed-double":      {previsNum("width", 0.2, 20, 1.8, false), previsNum("height", 0.2, 20, 1.05, false), previsNum("depth", 0.2, 20, 2.1, false), previsNum("surfaceHeight", 0.2, 20, 0.52, false)},
	"shotloom-furniture-bunk":            {previsNum("width", 0.2, 20, 1.05, false), previsNum("height", 0.2, 20, 1.95, false), previsNum("depth", 0.2, 20, 2.1, false), previsNum("surfaceHeight", 0.2, 20, 0.45, false)},
	"shotloom-furniture-crib":            {previsNum("width", 0.2, 20, 0.72, false), previsNum("height", 0.2, 20, 1.05, false), previsNum("depth", 0.2, 20, 1.32, false), previsNum("surfaceHeight", 0.2, 20, 0.55, false)},
	"shotloom-furniture-nightstand":      {previsNum("width", 0.2, 20, 0.48, false), previsNum("height", 0.2, 20, 0.55, false), previsNum("depth", 0.2, 20, 0.42, false)},
	"shotloom-furniture-vanity":          {previsNum("width", 0.2, 20, 1, false), previsNum("height", 0.2, 20, 1.5, false), previsNum("depth", 0.2, 20, 0.48, false), previsNum("surfaceHeight", 0.2, 20, 0.74, false)},
	"shotloom-furniture-screen":          {previsNum("width", 0.2, 20, 1.8, false), previsNum("height", 0.2, 20, 1.8, false), previsNum("depth", 0.2, 20, 0.3, false)},
	"shotloom-furniture-coatstand":       {previsNum("width", 0.2, 20, 0.6, false), previsNum("height", 0.2, 20, 1.75, false), previsNum("depth", 0.2, 20, 0.5, false)},
	"shotloom-furniture-cabinet":         {previsNum("width", 0.2, 20, 1.2, false), previsNum("height", 0.2, 20, 0.88, false), previsNum("depth", 0.2, 20, 0.6, false)},
	"shotloom-furniture-sink":            {previsNum("width", 0.2, 20, 0.9, false), previsNum("height", 0.2, 20, 0.92, false), previsNum("depth", 0.2, 20, 0.6, false)},
	"shotloom-furniture-stove":           {previsNum("width", 0.2, 20, 0.75, false), previsNum("height", 0.2, 20, 0.9, false), previsNum("depth", 0.2, 20, 0.6, false)},
	"shotloom-furniture-fridge":          {previsNum("width", 0.2, 20, 0.72, false), previsNum("height", 0.2, 20, 1.8, false), previsNum("depth", 0.2, 20, 0.7, false)},
	"shotloom-furniture-washer":          {previsNum("width", 0.2, 20, 0.62, false), previsNum("height", 0.2, 20, 0.87, false), previsNum("depth", 0.2, 20, 0.62, false)},
	"shotloom-furniture-toilet":          {previsNum("width", 0.2, 20, 0.42, false), previsNum("height", 0.2, 20, 0.82, false), previsNum("depth", 0.2, 20, 0.7, false)},
	"shotloom-furniture-bathtub":         {previsNum("width", 0.2, 20, 0.8, false), previsNum("height", 0.2, 20, 0.58, false), previsNum("depth", 0.2, 20, 1.7, false)},
	"shotloom-furniture-shower":          {previsNum("width", 0.2, 20, 0.9, false), previsNum("height", 0.2, 20, 2, false), previsNum("depth", 0.2, 20, 0.9, false)},
	"shotloom-furniture-partition":       {previsNum("width", 0.2, 20, 1.4, false), previsNum("height", 0.2, 20, 1.4, false), previsNum("depth", 0.2, 20, 0.3, false)},
	"shotloom-furniture-monitor":         {previsNum("width", 0.2, 20, 0.55, false), previsNum("height", 0.2, 20, 0.42, false), previsNum("depth", 0.2, 20, 0.2, false)},
	"shotloom-furniture-register":        {previsNum("width", 0.2, 20, 1.4, false), previsNum("height", 0.2, 20, 1.15, false), previsNum("depth", 0.2, 20, 0.65, false), previsNum("surfaceHeight", 0.2, 20, 0.9, false)},
	"shotloom-furniture-lectern":         {previsNum("width", 0.2, 20, 0.8, false), previsNum("height", 0.2, 20, 1.1, false), previsNum("depth", 0.2, 20, 0.55, false)},
	"shotloom-furniture-blackboard":      {previsNum("width", 0.2, 20, 2, false), previsNum("height", 0.2, 20, 1.8, false), previsNum("depth", 0.2, 20, 0.45, false)},
	"shotloom-furniture-hospitalbed":     {previsNum("width", 0.2, 20, 0.95, false), previsNum("height", 0.2, 20, 1.1, false), previsNum("depth", 0.2, 20, 2.1, false), previsNum("surfaceHeight", 0.2, 20, 0.68, false)},
	"shotloom-furniture-waitingbench":    {previsNum("width", 0.2, 20, 1.8, false), previsNum("height", 0.2, 20, 0.88, false), previsNum("depth", 0.2, 20, 0.58, false), previsNum("surfaceHeight", 0.2, 20, 0.46, false)},
	"shotloom-furniture-vending":         {previsNum("width", 0.2, 20, 1, false), previsNum("height", 0.2, 20, 1.85, false), previsNum("depth", 0.2, 20, 0.8, false)},
	"shotloom-prop-cup":                  {previsNum("width", 0.002, 20, 0.1, false), previsNum("height", 0.002, 20, 0.1, false), previsNum("depth", 0.002, 20, 0.08, false)},
	"shotloom-prop-bowl":                 {previsNum("width", 0.002, 20, 0.18, false), previsNum("height", 0.002, 20, 0.09, false), previsNum("depth", 0.002, 20, 0.18, false)},
	"shotloom-prop-plate":                {previsNum("width", 0.002, 20, 0.25, false), previsNum("height", 0.002, 20, 0.025, false), previsNum("depth", 0.002, 20, 0.25, false)},
	"shotloom-prop-bottle":               {previsNum("width", 0.002, 20, 0.075, false), previsNum("height", 0.002, 20, 0.28, false), previsNum("depth", 0.002, 20, 0.075, false)},
	"shotloom-prop-book":                 {previsNum("width", 0.002, 20, 0.16, false), previsNum("height", 0.002, 20, 0.035, false), previsNum("depth", 0.002, 20, 0.24, false)},
	"shotloom-prop-phone":                {previsNum("width", 0.002, 20, 0.075, false), previsNum("height", 0.002, 20, 0.008, false), previsNum("depth", 0.002, 20, 0.155, false)},
	"shotloom-prop-suitcase":             {previsNum("width", 0.002, 20, 0.4, false), previsNum("height", 0.002, 20, 0.6, false), previsNum("depth", 0.002, 20, 0.25, false)},
	"shotloom-prop-backpack":             {previsNum("width", 0.002, 20, 0.3, false), previsNum("height", 0.002, 20, 0.48, false), previsNum("depth", 0.002, 20, 0.2, false)},
	"shotloom-structure-wall":            {previsNum("width", 0.02, 100, 4, false), previsNum("height", 0.02, 100, 3, false), previsNum("depth", 0.02, 100, 0.18, false), previsNum("openingWidth", 0, 100, 0, false), previsNum("openingHeight", 0.02, 100, 2.1, false), previsNum("sill", 0, 100, 0, false), previsNum("openingOffset", -100, 100, 0, false)},
	"shotloom-structure-doorframe":       {previsNum("width", 0.02, 100, 1.1, false), previsNum("height", 0.02, 100, 2.2, false), previsNum("depth", 0.02, 100, 0.18, false), previsNum("frame", 0.02, 100, 0.08, false)},
	"shotloom-structure-door":            {previsNum("width", 0.02, 100, 1.1, false), previsNum("height", 0.02, 100, 2.2, false), previsNum("depth", 0.02, 100, 0.18, false), previsNum("frame", 0.02, 100, 0.08, false), previsNum("opening", 0, 160, 0, false)},
	"shotloom-structure-windowframe":     {previsNum("width", 0.02, 100, 1.5, false), previsNum("height", 0.02, 100, 1.2, false), previsNum("depth", 0.02, 100, 0.16, false), previsNum("frame", 0.02, 100, 0.08, false)},
	"shotloom-structure-window":          {previsNum("width", 0.02, 100, 1.5, false), previsNum("height", 0.02, 100, 1.2, false), previsNum("depth", 0.02, 100, 0.16, false), previsNum("frame", 0.02, 100, 0.08, false), previsNum("opening", 0, 160, 0, false)},
	"shotloom-structure-column":          {previsNum("width", 0.02, 100, 0.4, false), previsNum("height", 0.02, 100, 3, false), previsNum("depth", 0.02, 100, 0.4, false)},
	"shotloom-structure-beam":            {previsNum("width", 0.02, 100, 4, false), previsNum("height", 0.02, 100, 0.35, false), previsNum("depth", 0.02, 100, 0.3, false)},
	"shotloom-structure-slab":            {previsNum("width", 0.02, 100, 4, false), previsNum("height", 0.02, 100, 0.18, false), previsNum("depth", 0.02, 100, 4, false)},
	"shotloom-structure-stairs-straight": {previsNum("width", 0.02, 100, 1.2, false), previsNum("steps", 1, 128, 8, true), previsNum("rise", 0.02, 100, 0.18, false), previsNum("tread", 0.02, 100, 0.28, false), previsNum("landing", 0.02, 100, 1.2, false)},
	"shotloom-structure-stairs-l":        {previsNum("width", 0.02, 100, 1.2, false), previsNum("steps", 1, 128, 8, true), previsNum("rise", 0.02, 100, 0.18, false), previsNum("tread", 0.02, 100, 0.28, false), previsNum("landing", 0.02, 100, 1.2, false)},
	"shotloom-structure-stairs-u":        {previsNum("width", 0.02, 100, 1.2, false), previsNum("steps", 1, 128, 8, true), previsNum("rise", 0.02, 100, 0.18, false), previsNum("tread", 0.02, 100, 0.28, false), previsNum("landing", 0.02, 100, 1.2, false)},
	"shotloom-structure-stairs-crest":    {previsNum("width", 0.02, 100, 1.2, false), previsNum("steps", 1, 128, 8, true), previsNum("rise", 0.02, 100, 0.18, false), previsNum("tread", 0.02, 100, 0.28, false), previsNum("landing", 0.02, 100, 1.2, false)},
	"shotloom-structure-stairs-spiral":   {previsNum("radius", 0.02, 100, 1.5, false), previsNum("width", 0.02, 100, 1, false), previsNum("steps", 1, 128, 18, true), previsNum("rise", 0.02, 100, 0.18, false), previsNum("turn", 30, 1080, 300, false)},
	"shotloom-structure-ramp":            {previsNum("width", 0.02, 100, 1.5, false), previsNum("height", 0.02, 100, 1, false), previsNum("length", 0.02, 100, 5, false)},
	"shotloom-structure-ladder":          {previsNum("width", 0.2, 100, 0.5, false), previsNum("height", 0.4, 100, 3, false), previsNum("steps", 2, 64, 10, true), previsNum("tilt", 0, 50, 12, false)},
	"shotloom-structure-railing":         {previsNum("length", 0.2, 100, 3, false), previsNum("height", 0.2, 100, 1.1, false), previsNum("posts", 2, 128, 7, true)},
	"shotloom-road-straight":             {previsNum("width", 0.02, 100, 6, false), previsNum("height", 0.02, 100, 0.12, false), previsNum("length", 0.02, 100, 10, false)},
	"shotloom-road-curve":                {previsNum("width", 0.02, 100, 6, false), previsNum("height", 0.02, 100, 0.12, false), previsNum("radius", 0.02, 100, 8, false), previsNum("turn", 15, 270, 90, false)},
	"shotloom-road-junction":             {previsNum("width", 0.02, 100, 6, false), previsNum("height", 0.02, 100, 0.12, false), previsNum("length", 0.02, 100, 16, false)},
	"shotloom-road-sidewalk":             {previsNum("width", 0.02, 100, 2, false), previsNum("height", 0.02, 100, 0.18, false), previsNum("length", 0.02, 100, 8, false)},
	"shotloom-road-curb":                 {previsNum("width", 0.02, 100, 0.2, false), previsNum("height", 0.02, 100, 0.18, false), previsNum("length", 0.02, 100, 4, false)},
	"shotloom-road-bridge":               {previsNum("width", 0.02, 100, 5, false), previsNum("height", 0.02, 100, 0.35, false), previsNum("length", 0.02, 100, 10, false)},
	"shotloom-road-busstop":              {previsNum("width", 0.02, 100, 4, false), previsNum("height", 1.2, 100, 2.5, false), previsNum("length", 0.02, 100, 1.8, false)},
	"shotloom-road-parking":              {previsNum("width", 0.02, 100, 2.5, false), previsNum("height", 0.02, 100, 0.08, false), previsNum("length", 0.02, 100, 5, false)},
	"shotloom-plant-broadleaf":           {previsNum("width", 0.02, 100, 3.5, false), previsNum("height", 0.02, 100, 5, false), previsNum("depth", 0.02, 100, 3.5, false), previsNum("density", 4, 48, 12, true), previsNum("seed", 0, 2147483647, 1, true)},
	"shotloom-plant-conifer":             {previsNum("width", 0.02, 100, 2.6, false), previsNum("height", 0.02, 100, 6, false), previsNum("depth", 0.02, 100, 2.6, false), previsNum("density", 4, 48, 12, true), previsNum("seed", 0, 2147483647, 1, true)},
	"shotloom-plant-palm":                {previsNum("width", 0.02, 100, 4, false), previsNum("height", 0.02, 100, 6, false), previsNum("depth", 0.02, 100, 4, false), previsNum("density", 4, 48, 12, true), previsNum("seed", 0, 2147483647, 1, true)},
	"shotloom-plant-dead":                {previsNum("width", 0.02, 100, 3, false), previsNum("height", 0.02, 100, 4.5, false), previsNum("depth", 0.02, 100, 3, false), previsNum("density", 4, 48, 12, true), previsNum("seed", 0, 2147483647, 1, true)},
	"shotloom-plant-bush":                {previsNum("width", 0.02, 100, 1.4, false), previsNum("height", 0.02, 100, 1.2, false), previsNum("depth", 0.02, 100, 1.2, false), previsNum("density", 4, 48, 12, true), previsNum("seed", 0, 2147483647, 1, true)},
	"shotloom-plant-hedge":               {previsNum("width", 0.02, 100, 4, false), previsNum("height", 0.02, 100, 1.5, false), previsNum("depth", 0.02, 100, 0.7, false), previsNum("density", 4, 48, 12, true), previsNum("seed", 0, 2147483647, 1, true)},
	"shotloom-plant-grass":               {previsNum("width", 0.02, 100, 0.8, false), previsNum("height", 0.02, 100, 0.5, false), previsNum("depth", 0.02, 100, 0.8, false), previsNum("density", 4, 48, 24, true), previsNum("seed", 0, 2147483647, 1, true)},
	"shotloom-plant-potted":              {previsNum("width", 0.02, 100, 0.65, false), previsNum("height", 0.02, 100, 1, false), previsNum("depth", 0.02, 100, 0.65, false), previsNum("density", 4, 48, 12, true), previsNum("seed", 0, 2147483647, 1, true)},
	"shotloom-terrain-ground":            {previsNum("width", 0.02, 100, 10, false), previsNum("height", 0.02, 100, 0.2, false), previsNum("depth", 0.02, 100, 10, false)},
	"shotloom-terrain-slope":             {previsNum("width", 0.02, 100, 5, false), previsNum("height", 0.02, 100, 2, false), previsNum("depth", 0.02, 100, 5, false)},
	"shotloom-terrain-hill":              {previsNum("width", 0.02, 100, 8, false), previsNum("height", 0.02, 100, 3, false), previsNum("depth", 0.02, 100, 8, false), previsNum("roughness", 0, 0.7, 0.2, false), previsNum("seed", 0, 2147483647, 1, true)},
	"shotloom-terrain-rock":              {previsNum("width", 0.02, 100, 2, false), previsNum("height", 0.02, 100, 1.5, false), previsNum("depth", 0.02, 100, 1.6, false), previsNum("roughness", 0, 0.7, 0.2, false), previsNum("seed", 0, 2147483647, 1, true)},
	"shotloom-terrain-cliff":             {previsNum("width", 0.02, 100, 8, false), previsNum("height", 0.02, 100, 5, false), previsNum("depth", 0.02, 100, 2, false), previsNum("roughness", 0, 0.7, 0.2, false), previsNum("seed", 0, 2147483647, 1, true)},
	"shotloom-terrain-ditch":             {previsNum("width", 0.02, 100, 4, false), previsNum("height", 0.02, 100, 1.2, false), previsNum("depth", 0.02, 100, 8, false)},
	"shotloom-terrain-riverbed":          {previsNum("width", 0.02, 100, 8, false), previsNum("height", 0.02, 100, 1, false), previsNum("depth", 0.02, 100, 12, false), previsNum("roughness", 0, 0.7, 0.2, false), previsNum("seed", 0, 2147483647, 1, true)},
	"shotloom-terrain-water":             {previsNum("width", 0.02, 100, 8, false), previsNum("height", 0.02, 100, 0.02, false), previsNum("depth", 0.02, 100, 12, false)},
	"shotloom-building-bungalow":         {previsNum("width", 2, 100, 6, false), previsNum("depth", 1, 100, 5, false), previsNum("floorHeight", 2, 10, 3, false), previsNum("levels", 1, 16, 1, true), previsNum("bays", 1, 16, 3, true), previsNum("thickness", 0.05, 1, 0.18, false), previsNum("roofHeight", 0, 5, 1, false), previsEnum("roofStyle", "1", "0", "2", "3")},
	"shotloom-building-house":            {previsNum("width", 2, 100, 7, false), previsNum("depth", 1, 100, 6, false), previsNum("floorHeight", 2, 10, 3, false), previsNum("levels", 1, 16, 2, true), previsNum("bays", 1, 16, 3, true), previsNum("thickness", 0.05, 1, 0.18, false), previsNum("roofHeight", 0, 5, 1, false), previsEnum("roofStyle", "1", "0", "2", "3")},
	"shotloom-building-apartment":        {previsNum("width", 2, 100, 10, false), previsNum("depth", 1, 100, 8, false), previsNum("floorHeight", 2, 10, 3, false), previsNum("levels", 1, 16, 5, true), previsNum("bays", 1, 16, 3, true), previsNum("thickness", 0.05, 1, 0.18, false), previsNum("roofHeight", 0, 5, 1, false), previsEnum("roofStyle", "0", "1", "2", "3")},
	"shotloom-building-shop":             {previsNum("width", 2, 100, 5, false), previsNum("depth", 1, 100, 6, false), previsNum("floorHeight", 2, 10, 3, false), previsNum("levels", 1, 16, 1, true), previsNum("bays", 1, 16, 3, true), previsNum("thickness", 0.05, 1, 0.18, false), previsNum("roofHeight", 0, 5, 1, false), previsEnum("roofStyle", "0", "1", "2", "3")},
	"shotloom-building-warehouse":        {previsNum("width", 2, 100, 12, false), previsNum("depth", 1, 100, 8, false), previsNum("floorHeight", 2, 10, 4.5, false), previsNum("levels", 1, 16, 1, true), previsNum("bays", 1, 16, 3, true), previsNum("thickness", 0.05, 1, 0.18, false), previsNum("roofHeight", 0, 5, 1, false), previsEnum("roofStyle", "1", "0", "2", "3")},
	"shotloom-building-pavilion":         {previsNum("width", 2, 100, 4, false), previsNum("depth", 1, 100, 4, false), previsNum("floorHeight", 2, 10, 3, false), previsNum("levels", 1, 16, 1, true), previsNum("thickness", 0.05, 1, 0.18, false), previsNum("roofHeight", 0, 5, 1, false), previsEnum("roofStyle", "2", "0", "1", "3")},
	"shotloom-building-fortwall":         {previsNum("width", 2, 100, 10, false), previsNum("depth", 1, 100, 2.5, false), previsNum("floorHeight", 2, 10, 3, false), previsNum("levels", 1, 16, 1, true), previsNum("bays", 1, 16, 3, true), previsNum("thickness", 0.05, 1, 0.18, false), previsNum("roofHeight", 0, 5, 0.65, false), previsEnum("roofStyle", "3", "0")},
	"shotloom-building-tower":            {previsNum("width", 2, 100, 4, false), previsNum("depth", 1, 100, 4, false), previsNum("floorHeight", 2, 10, 3, false), previsNum("levels", 1, 16, 3, true), previsNum("bays", 1, 16, 3, true), previsNum("thickness", 0.05, 1, 0.18, false), previsNum("roofHeight", 0, 5, 0.65, false), previsEnum("roofStyle", "3", "0", "1", "2")},
	"shotloom-industrial-container":      {previsNum("width", 0.1, 100, 2.44, false), previsNum("height", 0.05, 100, 2.59, false), previsNum("depth", 0.1, 100, 6.06, false), previsNum("opening", 0, 150, 0, false)},
	"shotloom-industrial-pallet":         {previsNum("width", 0.1, 100, 1.2, false), previsNum("height", 0.05, 100, 0.15, false), previsNum("depth", 0.1, 100, 1, false)},
	"shotloom-industrial-barrel":         {previsNum("width", 0.1, 100, 0.6, false), previsNum("height", 0.05, 100, 0.9, false), previsNum("depth", 0.1, 100, 0.6, false)},
	"shotloom-industrial-pipeline":       {previsNum("width", 0.1, 100, 3, false), previsNum("height", 0.05, 100, 0.5, false), previsNum("depth", 0.1, 100, 0.5, false)},
	"shotloom-industrial-scaffold":       {previsNum("width", 0.1, 100, 2, false), previsNum("height", 0.05, 100, 6, false), previsNum("depth", 0.1, 100, 1.2, false), previsNum("levels", 1, 12, 2, true)},
	"shotloom-industrial-barrier":        {previsNum("width", 0.1, 100, 3, false), previsNum("height", 0.05, 100, 2, false), previsNum("depth", 0.1, 100, 0.25, false)},
	"shotloom-industrial-cone":           {previsNum("width", 0.1, 100, 0.4, false), previsNum("height", 0.05, 100, 0.7, false), previsNum("depth", 0.1, 100, 0.4, false)},
	"shotloom-industrial-handcart":       {previsNum("width", 0.1, 100, 0.7, false), previsNum("height", 0.05, 100, 1.1, false), previsNum("depth", 0.1, 100, 1.2, false)},
	"shotloom-theme-throne":              {previsNum("width", 0.1, 100, 1.1, false), previsNum("height", 0.05, 100, 1.8, false), previsNum("depth", 0.1, 100, 0.9, false)},
	"shotloom-theme-table":               {previsNum("width", 0.1, 100, 2, false), previsNum("height", 0.05, 100, 0.8, false), previsNum("depth", 0.1, 100, 0.8, false)},
	"shotloom-theme-weaponrack":          {previsNum("width", 0.1, 100, 1.8, false), previsNum("height", 0.05, 100, 1.8, false), previsNum("depth", 0.1, 100, 0.6, false), previsNum("slots", 1, 16, 5, true)},
	"shotloom-theme-flagpole":            {previsNum("width", 0.1, 100, 1.4, false), previsNum("height", 0.05, 100, 4, false), previsNum("depth", 0.1, 100, 0.5, false), previsEnum("flag", "1", "0")},
	"shotloom-theme-altar":               {previsNum("width", 0.1, 100, 3, false), previsNum("height", 0.05, 100, 1.3, false), previsNum("depth", 0.1, 100, 2.4, false), previsNum("steps", 1, 12, 3, true)},
	"shotloom-theme-ring":                {previsNum("width", 0.1, 100, 5, false), previsNum("height", 0.05, 100, 1.6, false), previsNum("depth", 0.1, 100, 5, false), previsEnum("ropes", "1", "0")},
	"shotloom-theme-tent":                {previsNum("width", 0.1, 100, 3, false), previsNum("height", 0.05, 100, 2.5, false), previsNum("depth", 0.1, 100, 4, false)},
	"shotloom-theme-stall":               {previsNum("width", 0.1, 100, 2, false), previsNum("height", 0.05, 100, 2.4, false), previsNum("depth", 0.1, 100, 1.2, false), previsEnum("canopy", "1", "0")},
	"shotloom-vehicle-sedan":             {previsNum("width", 0.1, 100, 1.8, false), previsNum("height", 0.05, 100, 1.45, false), previsNum("depth", 0.1, 100, 4.5, false), previsNum("seats", 1, 8, 4, true), previsNum("wheelRatio", 0.08, 0.35, 0.18, false), previsNum("wheelbaseRatio", 0.8, 1.2, 1, false), previsNum("trackRatio", 0.8, 1.15, 1, false)},
	"shotloom-vehicle-suv":               {previsNum("width", 0.1, 100, 1.95, false), previsNum("height", 0.05, 100, 1.8, false), previsNum("depth", 0.1, 100, 4.8, false), previsNum("seats", 1, 8, 4, true), previsNum("wheelRatio", 0.08, 0.35, 0.18, false), previsNum("wheelbaseRatio", 0.8, 1.2, 1, false), previsNum("trackRatio", 0.8, 1.15, 1, false)},
	"shotloom-vehicle-van":               {previsNum("width", 0.1, 100, 1.8, false), previsNum("height", 0.05, 100, 2, false), previsNum("depth", 0.1, 100, 4.6, false), previsNum("seats", 1, 8, 6, true), previsNum("wheelRatio", 0.08, 0.35, 0.18, false), previsNum("wheelbaseRatio", 0.8, 1.2, 1, false), previsNum("trackRatio", 0.8, 1.15, 1, false)},
	"shotloom-vehicle-bus":               {previsNum("width", 0.1, 100, 2.5, false), previsNum("height", 0.05, 100, 3.1, false), previsNum("depth", 0.1, 100, 10, false), previsNum("seats", 1, 40, 16, true), previsNum("wheelRatio", 0.08, 0.35, 0.18, false), previsNum("wheelbaseRatio", 0.8, 1.2, 1, false), previsNum("trackRatio", 0.8, 1.15, 1, false)},
	"shotloom-vehicle-truck":             {previsNum("width", 0.1, 100, 2.5, false), previsNum("height", 0.05, 100, 3.2, false), previsNum("depth", 0.1, 100, 7, false), previsNum("seats", 1, 8, 2, true), previsNum("wheelRatio", 0.08, 0.35, 0.18, false), previsNum("wheelbaseRatio", 0.8, 1.2, 1, false), previsNum("trackRatio", 0.8, 1.15, 1, false)},
	"shotloom-vehicle-bicycle":           {previsNum("width", 0.1, 100, 0.6, false), previsNum("height", 0.05, 100, 1.1, false), previsNum("depth", 0.1, 100, 1.8, false), previsNum("wheelRatio", 0.08, 0.35, 0.28, false), previsNum("wheelbaseRatio", 0.8, 1.2, 1, false)},
	"shotloom-vehicle-motorcycle":        {previsNum("width", 0.1, 100, 0.8, false), previsNum("height", 0.05, 100, 1.3, false), previsNum("depth", 0.1, 100, 2.1, false), previsNum("wheelRatio", 0.08, 0.35, 0.28, false), previsNum("wheelbaseRatio", 0.8, 1.2, 1, false)},
	"shotloom-vehicle-boat":              {previsNum("width", 0.1, 100, 1.5, false), previsNum("height", 0.05, 100, 0.8, false), previsNum("depth", 0.1, 100, 3.6, false), previsNum("seats", 1, 8, 3, true)},
	"building": {previsNum("floors", 1, 80, 6, true), previsNum("width", 4, 80, 14, false), previsNum("depth", 4, 60, 12, false), previsNum("floorHeight", 2.6, 6, 3.2, false),
		previsEnum("style", "modern", "glass", "brick", "concrete", "ancient"), previsEnum("roof", "flat", "gable", "chinese"), previsEnum("balcony", "none", "front")},
	"tower":        {previsNum("floors", 10, 150, 40, true), previsNum("width", 12, 70, 28, false), previsNum("setbacks", 0, 3, 2, true), previsEnum("crown", "spire", "flat", "crown")},
	"chinese_hall": {previsNum("bays", 3, 11, 5, true), previsNum("depth", 5, 20, 9, false), previsNum("tiers", 1, 5, 1, true)},
	"road":         {previsNum("length", 2, 600, 40, false), previsNum("width", 2, 60, 10, false), previsNum("lanes", 0, 8, 2, true), previsNum("sidewalk", 0, 10, 2.5, false), previsEnum("surface", "asphalt", "stone", "dirt")},
	"wall":         {previsNum("length", 0.5, 200, 8, false), previsNum("height", 0.3, 20, 3, false), previsNum("thickness", 0.08, 8, 0.3, false), previsEnum("material", previsMaterials...), previsEnum("top", "none", "tiles", "battlement")},
	"ground_patch": {previsNum("width", 0.5, 800, 20, false), previsNum("depth", 0.5, 800, 20, false), previsEnum("surface", "grass", "plaza", "stone", "sand", "water", "soil", "snow")},
	"stairs":       {previsNum("width", 0.6, 40, 3, false), previsNum("steps", 1, 60, 8, true), previsNum("stepHeight", 0.08, 0.5, 0.17, false), previsNum("stepDepth", 0.15, 1, 0.3, false), previsEnum("material", "stone", "concrete", "wood")},
	"fence_line":   {previsNum("length", 1, 200, 8, false), previsNum("height", 0.4, 4, 1.1, false), previsEnum("style", "wood", "metal", "stone")},
	"room":         {previsNum("width", 2, 40, 6, false), previsNum("depth", 2, 40, 5, false), previsNum("height", 2.2, 8, 3, false), previsNum("windows", 0, 6, 2, true), previsEnum("door", "right", "none", "left", "back"), previsEnum("front", "open", "closed")},
	"platform":     {previsNum("width", 0.5, 200, 8, false), previsNum("depth", 0.5, 200, 8, false), previsNum("height", 0.05, 10, 0.6, false), previsEnum("material", "stone", "concrete", "wood", "plaster")},
	"column":       {previsNum("height", 0.5, 30, 3, false), previsNum("radius", 0.05, 3, 0.25, false), previsEnum("style", "round", "square", "chinese")},
}

// 目录道具默认占地（宽深取大值，米），用于批量排布的默认间距；数据由前端目录生成。
var cloudAgentPrevisFootprints = map[string]float64{
	"shotloom-shape-box":                 1,
	"shotloom-shape-sphere":              1,
	"shotloom-shape-cylinder":            1,
	"shotloom-shape-cone":                1,
	"shotloom-shape-capsule":             1,
	"shotloom-shape-torus":               1,
	"shotloom-shape-pyramid":             1,
	"shotloom-shape-plane":               1,
	"shotloom-shape-wedge":               1,
	"shotloom-shape-ramp":                1,
	"shotloom-shape-arch":                1,
	"shotloom-shape-hemisphere":          1,
	"shotloom-shape-tube":                1,
	"shotloom-shape-l":                   1,
	"shotloom-shape-u":                   1,
	"shotloom-shape-arc":                 1,
	"shotloom-furniture-chair":           0.5,
	"shotloom-furniture-stool":           0.38,
	"shotloom-furniture-bench":           1.6,
	"shotloom-furniture-armchair":        0.65,
	"shotloom-furniture-sofa-single":     0.92,
	"shotloom-furniture-sofa":            2.2,
	"shotloom-furniture-barstool":        0.43,
	"shotloom-furniture-lounger":         1.81106,
	"shotloom-furniture-table":           1.4,
	"shotloom-furniture-roundtable":      1.2,
	"shotloom-furniture-desk":            1.3,
	"shotloom-furniture-coffeetable":     1.05,
	"shotloom-furniture-counter":         2,
	"shotloom-furniture-wardrobe":        1.5,
	"shotloom-furniture-bookshelf":       0.9,
	"shotloom-furniture-rack":            1.2,
	"shotloom-furniture-bed-single":      2.05,
	"shotloom-furniture-bed-double":      2.1,
	"shotloom-furniture-bunk":            2.1,
	"shotloom-furniture-crib":            1.32,
	"shotloom-furniture-nightstand":      0.48,
	"shotloom-furniture-vanity":          1,
	"shotloom-furniture-screen":          1.776,
	"shotloom-furniture-coatstand":       0.6,
	"shotloom-furniture-cabinet":         1.2,
	"shotloom-furniture-sink":            0.9,
	"shotloom-furniture-stove":           0.75,
	"shotloom-furniture-fridge":          0.745,
	"shotloom-furniture-washer":          0.6875,
	"shotloom-furniture-toilet":          0.616,
	"shotloom-furniture-bathtub":         1.7,
	"shotloom-furniture-shower":          0.9,
	"shotloom-furniture-partition":       1.376,
	"shotloom-furniture-monitor":         0.55,
	"shotloom-furniture-register":        1.4,
	"shotloom-furniture-lectern":         0.8,
	"shotloom-furniture-blackboard":      1.964,
	"shotloom-furniture-hospitalbed":     2.1,
	"shotloom-furniture-waitingbench":    1.8,
	"shotloom-furniture-vending":         1,
	"shotloom-prop-cup":                  0.1,
	"shotloom-prop-bowl":                 0.18,
	"shotloom-prop-plate":                0.25,
	"shotloom-prop-bottle":               0.075,
	"shotloom-prop-book":                 0.24,
	"shotloom-prop-phone":                0.155,
	"shotloom-prop-suitcase":             0.4,
	"shotloom-prop-backpack":             0.3,
	"shotloom-structure-wall":            4,
	"shotloom-structure-doorframe":       1.1,
	"shotloom-structure-door":            1.1,
	"shotloom-structure-windowframe":     1.5,
	"shotloom-structure-window":          1.5,
	"shotloom-structure-column":          0.4,
	"shotloom-structure-beam":            4,
	"shotloom-structure-slab":            4,
	"shotloom-structure-stairs-straight": 2.24,
	"shotloom-structure-stairs-l":        3.44,
	"shotloom-structure-stairs-u":        3.44,
	"shotloom-structure-stairs-crest":    5.68,
	"shotloom-structure-stairs-spiral":   2.99999,
	"shotloom-structure-ramp":            5,
	"shotloom-structure-ladder":          0.67264,
	"shotloom-structure-railing":         3,
	"shotloom-road-straight":             10,
	"shotloom-road-curve":                11,
	"shotloom-road-junction":             16,
	"shotloom-road-sidewalk":             8,
	"shotloom-road-curb":                 4,
	"shotloom-road-bridge":               10,
	"shotloom-road-busstop":              4,
	"shotloom-road-parking":              5,
	"shotloom-plant-broadleaf":           3.5,
	"shotloom-plant-conifer":             2.6,
	"shotloom-plant-palm":                4,
	"shotloom-plant-dead":                3,
	"shotloom-plant-bush":                1.4,
	"shotloom-plant-hedge":               4,
	"shotloom-plant-grass":               0.8,
	"shotloom-plant-potted":              0.65,
	"shotloom-terrain-ground":            10,
	"shotloom-terrain-slope":             5,
	"shotloom-terrain-hill":              8,
	"shotloom-terrain-rock":              2,
	"shotloom-terrain-cliff":             8,
	"shotloom-terrain-ditch":             8,
	"shotloom-terrain-riverbed":          12,
	"shotloom-terrain-water":             12,
	"shotloom-building-bungalow":         6.3,
	"shotloom-building-house":            7.3,
	"shotloom-building-apartment":        10.2,
	"shotloom-building-shop":             6.2,
	"shotloom-building-warehouse":        12.3,
	"shotloom-building-pavilion":         4.4,
	"shotloom-building-fortwall":         10,
	"shotloom-building-tower":            4,
	"shotloom-industrial-container":      6.06,
	"shotloom-industrial-pallet":         1.2,
	"shotloom-industrial-barrel":         0.6,
	"shotloom-industrial-pipeline":       3,
	"shotloom-industrial-scaffold":       2,
	"shotloom-industrial-barrier":        3,
	"shotloom-industrial-cone":           0.4,
	"shotloom-industrial-handcart":       1.2,
	"shotloom-theme-throne":              1.1,
	"shotloom-theme-table":               2,
	"shotloom-theme-weaponrack":          1.8,
	"shotloom-theme-flagpole":            1.4,
	"shotloom-theme-altar":               3,
	"shotloom-theme-ring":                5,
	"shotloom-theme-tent":                4,
	"shotloom-theme-stall":               2,
	"shotloom-vehicle-sedan":             4.5,
	"shotloom-vehicle-suv":               4.8,
	"shotloom-vehicle-van":               4.6,
	"shotloom-vehicle-bus":               10,
	"shotloom-vehicle-truck":             7,
	"shotloom-vehicle-bicycle":           1.8,
	"shotloom-vehicle-motorcycle":        2.1,
	"shotloom-vehicle-boat":              3.6,
	"building":                           14, "tower": 28, "chinese_hall": 17, "road": 40, "wall": 8, "ground_patch": 20, "stairs": 3, "fence_line": 8, "room": 6, "platform": 8, "column": 0.7,
	"tree_broadleaf": 5, "tree_pine": 3.6, "tree_willow": 5.5, "tree_cherry": 4.6, "tree_palm": 4, "bamboo": 2.4, "bush": 1.6, "hedge": 4, "grass": 0.9, "flowers": 1.2, "flower_bed": 3, "potted_plant": 0.6, "lotus_pond": 6,
	"ancient_house": 9, "ancient_shop": 8, "ancient_courtyard": 22, "tower_hall": 16, "pagoda": 9, "pavilion": 5, "city_gate": 32, "ancient_wall": 20, "paifang": 12, "stone_bridge": 14, "well": 2.4, "market_stall": 3, "lantern_pole": 0.8,
	"modern_house": 11, "villa": 10, "apartment": 18, "office_tower": 26, "skyscraper": 32, "shop_front": 10,
	"street_lamp": 1.6, "traffic_light": 3.4, "bench": 1.8, "bus_stop": 5, "fence": 4, "trash_bin": 0.6,
	"car": 4.6, "bus": 11, "ox_cart": 5.4, "bicycle": 1.8, "rock": 1.4, "boulder": 4, "hill": 40, "log": 4,
	"sofa": 2.2, "dining_table": 1.6, "chair": 0.52, "bed": 2.15, "bookshelf": 1.2, "desk": 1.4,
}

// cloudAgentPrevisNormalizeParams 校验并补全参数：未知键、越界数值、非法枚举一律拒绝，缺省键填默认值。
// base 为已有参数（object_update 时合并），patch 为 Agent 本次提交的参数。
func cloudAgentPrevisNormalizeParams(propID string, base map[string]any, patch map[string]any) (result map[string]any, err error) {
	// 此函数只校验目录参数；明确标成可修正字段错误，不改变外层权限、归属和审批校验。
	defer func() {
		if err != nil {
			err = cloudAgentFieldError("params", "invalid_value", err.Error())
		}
	}()
	specs, ok := cloudAgentPrevisPropParams[propID]
	if !ok {
		if len(patch) > 0 {
			hint := "；只可修改 position/rotation/scale/color，参数化道具须从 catalog.params 选择"
			if propID == "car" {
				hint += "；可调轿车使用 propId=shotloom-vehicle-sedan"
			}
			return nil, BadAuthRequest(fmt.Sprintf("道具 %s 没有可调参数%s", propID, hint))
		}
		return nil, nil
	}
	byKey := make(map[string]cloudAgentPrevisParamSpec, len(specs))
	for _, spec := range specs {
		byKey[spec.Key] = spec
	}
	for key := range patch {
		if _, ok := byKey[key]; !ok {
			return nil, BadAuthRequest(fmt.Sprintf("道具 %s 不支持参数 %s，可用：%s", propID, key, cloudAgentPrevisParamKeys(propID)))
		}
	}
	result = make(map[string]any, len(specs))
	for _, spec := range specs {
		value, provided := patch[spec.Key]
		if !provided {
			value, provided = base[spec.Key]
		}
		if len(spec.Options) > 0 {
			text, _ := value.(string)
			if !provided {
				text = spec.Options[0]
			}
			if err := cloudAgentPrevisValidateEnum(text, "参数 "+spec.Key, spec.Options...); err != nil {
				return nil, err
			}
			result[spec.Key] = text
			continue
		}
		number := spec.Default
		if provided {
			number = numberValue(value, math.NaN())
		}
		if err := cloudAgentPrevisFiniteInRange(number, "参数 "+spec.Key, spec.Min, spec.Max); err != nil {
			return nil, err
		}
		if spec.Integer {
			number = math.Round(number)
		}
		result[spec.Key] = number
	}
	return result, nil
}

func cloudAgentPrevisParamKeys(propID string) string {
	keys := make([]string, 0)
	for _, spec := range cloudAgentPrevisPropParams[propID] {
		if len(spec.Options) > 0 {
			keys = append(keys, spec.Key+"("+strings.Join(spec.Options, "/")+")")
		} else {
			keys = append(keys, fmt.Sprintf("%s(%g-%g)", spec.Key, spec.Min, spec.Max))
		}
	}
	return strings.Join(keys, ",")
}

// cloudAgentPrevisParamsGuide 列出所有参数化组件的参数，放进 previs_scene_read.catalog。
func cloudAgentPrevisParamsGuide() map[string]string {
	ids := make([]string, 0, len(cloudAgentPrevisPropParams))
	for id := range cloudAgentPrevisPropParams {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	guide := make(map[string]string, len(ids))
	for _, id := range ids {
		guide[id] = cloudAgentPrevisParamKeys(id)
	}
	return guide
}
