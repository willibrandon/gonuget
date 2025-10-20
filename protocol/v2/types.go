// Package v2 implements the NuGet v2 OData protocol.
//
// It provides feed detection, package search, metadata access,
// and package download functionality for NuGet v2 feeds.
package v2

import (
	"encoding/xml"
)

// Service represents the OData service document.
type Service struct {
	XMLName   xml.Name  `xml:"service"`
	Workspace Workspace `xml:"workspace"`
	Base      string    `xml:"base,attr"`
}

// Workspace contains collections in the OData service.
type Workspace struct {
	Title       string       `xml:"title"`
	Collections []Collection `xml:"collection"`
}

// Collection represents an OData collection.
type Collection struct {
	Href  string `xml:"href,attr"`
	Title string `xml:"title"`
}

// Feed represents an Atom feed response.
type Feed struct {
	XMLName xml.Name `xml:"feed"`
	Title   string   `xml:"title"`
	ID      string   `xml:"id"`
	Updated string   `xml:"updated"`
	Entries []Entry  `xml:"entry"`
}

// Entry represents a single entry in an Atom feed.
type Entry struct {
	XMLName    xml.Name   `xml:"entry"`
	ID         string     `xml:"id"`
	Title      string     `xml:"title"`
	Updated    string     `xml:"updated"`
	Properties Properties `xml:"properties"`
	Content    Content    `xml:"content"`
}

// Properties contains package metadata.
type Properties struct {
	XMLName                  xml.Name `xml:"properties"`
	ID                       string   `xml:"Id"`
	Version                  string   `xml:"Version"`
	Description              string   `xml:"Description"`
	Authors                  string   `xml:"Authors"`
	IconURL                  string   `xml:"IconUrl"`
	LicenseURL               string   `xml:"LicenseUrl"`
	ProjectURL               string   `xml:"ProjectUrl"`
	Tags                     string   `xml:"Tags"`
	Dependencies             string   `xml:"Dependencies"`
	DownloadCount            int64    `xml:"DownloadCount"`
	IsPrerelease             bool     `xml:"IsPrerelease"`
	Published                string   `xml:"Published"`
	RequireLicenseAcceptance bool     `xml:"RequireLicenseAcceptance"`
}

// Content contains the package download URL.
type Content struct {
	Type string `xml:"type,attr"`
	Src  string `xml:"src,attr"`
}
