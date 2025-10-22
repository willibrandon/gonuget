package assets

import "github.com/willibrandon/gonuget/frameworks"

// ContentItemCollection manages package assets and performs selection.
// Reference: ContentModel/ContentItemCollection.cs
type ContentItemCollection struct {
	Assets []*ContentItem
}

// NewContentItemCollection creates a collection from file paths.
func NewContentItemCollection(paths []string) *ContentItemCollection {
	assets := make([]*ContentItem, len(paths))
	for i, path := range paths {
		assets[i] = &ContentItem{Path: path, Properties: make(map[string]any)}
	}
	return &ContentItemCollection{Assets: assets}
}

// PopulateItemGroups groups assets by their properties.
// Groups are created by matching assets against group patterns, then items are matched against path patterns.
// Reference: ContentItemCollection.cs PopulateItemGroups (Lines 84-134)
func (c *ContentItemCollection) PopulateItemGroups(patternSet *PatternSet) []*ContentItemGroup {
	if len(c.Assets) == 0 {
		return nil
	}

	groupAssets := make(map[string]*ContentItemGroup)

	for _, asset := range c.Assets {
		// Try each group pattern
		for _, groupExpr := range patternSet.GroupExpressions {
			item := groupExpr.Match(asset.Path, patternSet.PropertyDefinitions)
			if item != nil {
				// Create group key from properties
				groupKey := buildGroupKey(item.Properties)

				if _, exists := groupAssets[groupKey]; !exists {
					groupAssets[groupKey] = &ContentItemGroup{
						Properties: item.Properties,
						Items:      []*ContentItem{},
					}
				}

				// Find matching items using path patterns
				for _, pathExpr := range patternSet.PathExpressions {
					pathItem := pathExpr.Match(asset.Path, patternSet.PropertyDefinitions)
					if pathItem != nil {
						groupAssets[groupKey].Items = append(groupAssets[groupKey].Items, pathItem)
						break
					}
				}
				break
			}
		}
	}

	// Convert map to slice
	groups := make([]*ContentItemGroup, 0, len(groupAssets))
	for _, group := range groupAssets {
		if len(group.Items) > 0 {
			groups = append(groups, group)
		}
	}

	return groups
}

// FindBestItemGroup selects the best matching group for criteria.
// Tries each criteria entry in order, finding the best matching group using property comparison.
// Reference: ContentItemCollection.cs FindBestItemGroup (Lines 136-241)
func (c *ContentItemCollection) FindBestItemGroup(criteria *SelectionCriteria, patternSets ...*PatternSet) *ContentItemGroup {
	for _, patternSet := range patternSets {
		groups := c.PopulateItemGroups(patternSet)

		// Try each criteria entry in order
		for _, criteriaEntry := range criteria.Entries {
			var bestGroup *ContentItemGroup

			for _, itemGroup := range groups {
				groupIsValid := true

				// Check if group satisfies all criteria properties
				for key, criteriaValue := range criteriaEntry.Properties {
					if criteriaValue == nil {
						// Criteria requires property to NOT exist
						if _, exists := itemGroup.Properties[key]; exists {
							groupIsValid = false
							break
						}
					} else {
						// Criteria requires property to exist and be compatible
						itemValue, exists := itemGroup.Properties[key]
						if !exists {
							groupIsValid = false
							break
						}

						propDef, hasDef := patternSet.PropertyDefinitions[key]
						if !hasDef {
							groupIsValid = false
							break
						}

						// Use property definition's compatibility test
						if !propDef.IsCriteriaSatisfied(criteriaValue, itemValue) {
							groupIsValid = false
							break
						}
					}
				}

				if groupIsValid {
					if bestGroup == nil {
						bestGroup = itemGroup
					} else {
						// Compare groups to find better match
						groupComparison := 0

						for key, criteriaValue := range criteriaEntry.Properties {
							if criteriaValue == nil {
								continue
							}

							bestGroupValue := bestGroup.Properties[key]
							itemGroupValue := itemGroup.Properties[key]
							propDef := patternSet.PropertyDefinitions[key]

							groupComparison = propDef.Compare(criteriaValue, bestGroupValue, itemGroupValue)
							if groupComparison != 0 {
								break
							}
						}

						if groupComparison > 0 {
							// itemGroup is better
							bestGroup = itemGroup
						}
					}
				}
			}

			if bestGroup != nil {
				return bestGroup
			}
		}
	}

	return nil
}

// ContentItemGroup represents assets grouped by properties.
// Reference: ContentModel/ContentItemGroup.cs
type ContentItemGroup struct {
	Properties map[string]any
	Items      []*ContentItem
}

func buildGroupKey(properties map[string]any) string {
	// Build stable key from properties
	key := ""
	if tfm, ok := properties["tfm"]; ok {
		if fw, ok := tfm.(*frameworks.NuGetFramework); ok {
			key += fw.String() + "|"
		}
	}
	if rid, ok := properties["rid"]; ok {
		if ridStr, ok := rid.(string); ok {
			key += ridStr
		}
	}
	return key
}
