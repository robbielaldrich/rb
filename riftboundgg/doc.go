// Package riftboundgg mirrors the riftbound.gg card database onto local disk,
// in the shape package cards reads.
//
// riftbound.gg publishes no API of its own; its pages are fed by DotGG's card
// endpoint (https://api.dotgg.gg/cgfw/getcards?game=riftbound), which is
// undocumented and may change without notice.
package riftboundgg
