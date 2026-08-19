// Package index defines the Elasticsearch index names, settings and mappings
// used by the search subsystem, plus the document shapes that get indexed.
//
// Three indices back the public search API:
//
//	isilo_isps         -> ISP profiles          (search/isp, search/isp/{county})
//	isilo_technicians  -> Technician profiles   (search/tech, search/role, near-me)
//	isilo_locations    -> Administrative places (search/location, did-you-mean)
//
// Every text field that participates in autocomplete carries an
// `edge_ngram`-backed sub-field, and every "did you mean" field carries a
// `completion` suggester plus a keyword sub-field for exact/typo scoring.
package index

// Index name constants. Prefixed to avoid collisions on a shared cluster.
//declared in indices .go file
// sharedSettings defines custom analyzers reused across indices:
//
//   - autocomplete       : edge n-grams for as-you-type prefix matching.
//   - autocomplete_search: plain lowercase, used at query time so we don't
//     n-gram the search term itself (a classic edge-ngram mistake).
//   - folding            : lowercase + asciifolding so "Murang'a" matches
//     "muranga" and accents are ignored.
const sharedSettings = `
  "settings": {
    "index": {
      "number_of_shards": 1,
      "number_of_replicas": 1,
      "max_ngram_diff": 18
    },
    "analysis": {
      "filter": {
        "autocomplete_filter": {
          "type": "edge_ngram",
          "min_gram": 2,
          "max_gram": 20
        }
      },
      "analyzer": {
        "autocomplete": {
          "type": "custom",
          "tokenizer": "standard",
          "filter": ["lowercase", "asciifolding", "autocomplete_filter"]
        },
        "autocomplete_search": {
          "type": "custom",
          "tokenizer": "standard",
          "filter": ["lowercase", "asciifolding"]
        },
        "folding": {
          "type": "custom",
          "tokenizer": "standard",
          "filter": ["lowercase", "asciifolding"]
        }
      }
    }
  }`

// ISPMapping is the full create-index body for the ISP index.
var ISPMapping = `{
` + sharedSettings + `,
  "mappings": {
    "properties": {
      "id":            { "type": "keyword" },
      "name": {
        "type": "text",
        "analyzer": "folding",
        "fields": {
          "autocomplete": { "type": "text", "analyzer": "autocomplete", "search_analyzer": "autocomplete_search" },
          "keyword":      { "type": "keyword" },
          "suggest":      { "type": "completion" }
        }
      },
      "description":    { "type": "text", "analyzer": "folding" },
      "avatar_url":     { "type": "keyword", "index": false },
      "county":         { "type": "text", "analyzer": "folding", "fields": { "keyword": { "type": "keyword" } } },
      "sub_county":     { "type": "text", "analyzer": "folding", "fields": { "keyword": { "type": "keyword" } } },
      "village":        { "type": "text", "analyzer": "folding", "fields": { "keyword": { "type": "keyword" } } },
      "coverage_areas": { "type": "text", "analyzer": "folding", "fields": { "keyword": { "type": "keyword" } } },
      "rating":         { "type": "float" },
      "review_count":   { "type": "integer" },
      "is_active":      { "type": "boolean" },
      "location":       { "type": "geo_point" },
      "created_at":     { "type": "date" }
    }
  }
}`

// TechnicianMapping is the full create-index body for the technician index.
var TechnicianMapping = `{
` + sharedSettings + `,
  "mappings": {
    "properties": {
      "id":           { "type": "keyword" },
      "user_id":      { "type": "keyword" },
      "name": {
        "type": "text",
        "analyzer": "folding",
        "fields": {
          "autocomplete": { "type": "text", "analyzer": "autocomplete", "search_analyzer": "autocomplete_search" },
          "keyword":      { "type": "keyword" },
          "suggest":      { "type": "completion" }
        }
      },
      "avatar_url":   { "type": "keyword", "index": false },
      "isp_id":       { "type": "keyword" },
      "isp_name":     { "type": "text", "analyzer": "folding", "fields": { "keyword": { "type": "keyword" } } },
      "roles":        { "type": "keyword" },
      "skills":       { "type": "keyword" },
      "county":       { "type": "text", "analyzer": "folding", "fields": { "keyword": { "type": "keyword" } } },
      "sub_county":   { "type": "text", "analyzer": "folding", "fields": { "keyword": { "type": "keyword" } } },
      "village":      { "type": "text", "analyzer": "folding", "fields": { "keyword": { "type": "keyword" } } },
      "rating":       { "type": "float" },
      "jobs_done":    { "type": "integer" },
      "is_available": { "type": "boolean" },
      "location":     { "type": "geo_point" },
      "created_at":   { "type": "date" }
    }
  }
}`

// LocationMapping is the full create-index body for the administrative
// place index. `name.suggest` powers the completion suggester and
// `name.keyword` powers term-suggester based "did you mean".
var LocationMapping = `{
` + sharedSettings + `,
  "mappings": {
    "properties": {
      "id":         { "type": "keyword" },
      "name": {
        "type": "text",
        "analyzer": "folding",
        "fields": {
          "autocomplete": { "type": "text", "analyzer": "autocomplete", "search_analyzer": "autocomplete_search" },
          "keyword":      { "type": "keyword" },
          "suggest":      { "type": "completion" }
        }
      },
      "type":       { "type": "keyword" },
      "county":     { "type": "text", "analyzer": "folding", "fields": { "keyword": { "type": "keyword" } } },
      "sub_county": { "type": "text", "analyzer": "folding", "fields": { "keyword": { "type": "keyword" } } },
      "ward":       { "type": "text", "analyzer": "folding", "fields": { "keyword": { "type": "keyword" } } },
      "point":      { "type": "geo_point" },
      "created_at": { "type": "date" }
    }
  }
}`

// Mappings maps each index name to its create body, used by the index manager
// to ensure all indices exist on startup.
var Mappings = map[string]string{
	ISPIndex:        ISPMapping,
	TechnicianIndex: TechnicianMapping,
	LocationIndex:   LocationMapping,
}
