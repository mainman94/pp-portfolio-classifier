package ppxml

import (
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"pp-portfolio-classifier/internal/model"
	"pp-portfolio-classifier/internal/taxonomy"
)

type File struct {
	root       *Node
	path       string
	securities []*model.Security
}

func Load(path string) (*File, error) {
	root, err := ParseFile(path)
	if err != nil {
		return nil, err
	}
	if root.Attr("id") != "" {
		return nil, fmt.Errorf("xml format with id attributes is not supported")
	}
	return &File{root: root, path: path}, nil
}

func (f *File) Securities() ([]*model.Security, error) {
	if f.securities != nil {
		return f.securities, nil
	}
	secRoot := f.root.Child("securities")
	if secRoot == nil {
		return nil, fmt.Errorf("missing securities section")
	}
	var out []*model.Security
	for i, sec := range secRoot.ChildrenNamed("security") {
		coinID := ""
		for _, prop := range sec.ChildrenNamed("property") {
			if prop.Attr("name") == "COINGECKOCOINID" {
				coinID = strings.TrimSpace(prop.Text)
				break
			}
		}
		s := &model.Security{
			Name:      childText(sec, "name"),
			ISIN:      childText(sec, "isin"),
			Ticker:    childText(sec, "tickerSymbol"),
			Feed:      childText(sec, "feed"),
			CoinID:    coinID,
			UUID:      childText(sec, "uuid"),
			Note:      childText(sec, "note"),
			IsRetired: strings.EqualFold(childText(sec, "isRetired"), "true"),
			SourceRef: securityRef(i),
		}
		if s.Name == "" {
			continue
		}
		if strings.Contains(s.Note, "#PPC:SKIP") {
			continue
		}
		if s.ISIN == "" && !isCryptoCandidate(s) {
			continue
		}
		if match := regexp.MustCompile(`#PPC:\[ISIN2=([A-Z0-9]{12})`).FindStringSubmatch(s.Note); len(match) == 2 {
			s.AltISIN = match[1]
		}
		out = append(out, s)
	}
	f.securities = out
	return out, nil
}

func (f *File) UpdateTaxonomy(kind string, securities []*model.Security, topHoldings int, countryByRegion bool) error {
	taxonomyNode := f.ensureTaxonomy(kind)
	root := taxonomyNode.EnsureChild("root")
	children := root.EnsureChild("children")
	removeAssignmentsForSecurities(root, securities)

	categoryNodes := map[string]*Node{}
	if kind != "Country@Region" {
		for _, class := range children.ChildrenNamed("classification") {
			categoryNodes[childText(class, "name")] = class
		}
	}

	colorIndex := len(categoryNodes) % len(taxonomy.Colors)
	rankCounter := 1

	for _, security := range securities {
		report := mergedReport(security, kind)
		if report == nil {
			continue
		}
		assignments := report.Taxonomies[kind]
		if kind == "Holding" && topHoldings == 0 {
			continue
		}
		if len(assignments) == 0 {
			continue
		}

		scale := scaling(assignments)
		for name, rawWeight := range assignments {
			weight := int64(taxonomy.ValidateWeight(rawWeight*scale) * 100.0)
			if weight == 0 {
				continue
			}

			if kind == "Country@Region" {
				if !countryByRegion {
					continue
				}
				level1 := taxonomy.RegionForCountry(name)
				parent := ensureClassification(children, level1, &colorIndex)
				childClass := ensureClassification(parent.EnsureChild("children"), name, &colorIndex)
				ensureAssignment(childClass.EnsureChild("assignments"), nestedSecurityRef(security.SourceRef), weight, &rankCounter)
				continue
			}

			class := categoryNodes[name]
			if class == nil {
				class = newClassification(name, taxonomy.Colors[colorIndex%len(taxonomy.Colors)], len(categoryNodes))
				colorIndex++
				children.Append(class)
				categoryNodes[name] = class
			}
			ensureAssignment(class.EnsureChild("assignments"), security.SourceRef, weight, &rankCounter)
		}
	}

	removeZeroAssignments(root)
	return nil
}

func (f *File) DumpCSV(path string, securities []*model.Security) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := csv.NewWriter(file)
	defer w.Flush()
	if err := w.Write([]string{"ISIN", "Taxonomy", "Classification", "Percentage", "Name"}); err != nil {
		return err
	}
	for _, security := range securities {
		report := mergedReport(security, "")
		if report == nil {
			continue
		}
		for _, def := range taxonomy.Ordered {
			rows := report.Taxonomies[def.Name]
			var names []string
			for name := range rows {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				value := rows[name]
				if value <= 0 {
					continue
				}
				if err := w.Write([]string{
					security.ISIN,
					def.Name,
					name,
					strconv.FormatFloat(value/100.0, 'f', -1, 64),
					security.Name,
				}); err != nil {
					return err
				}
			}
		}
	}
	return w.Error()
}

func isCryptoCandidate(security *model.Security) bool {
	text := strings.ToLower(strings.Join([]string{
		security.Name, security.Ticker, security.Feed, security.CoinID,
	}, " "))
	if strings.Contains(text, "bitcoin") || strings.Contains(text, "ethereum") || strings.Contains(text, "crypto") ||
		strings.Contains(text, "blockchain") || strings.Contains(text, "solana") || strings.Contains(text, "xrp") ||
		strings.EqualFold(strings.TrimSpace(security.Feed), "COINGECKO") {
		return true
	}
	return false
}

func (f *File) Write(path string) error {
	return f.root.WriteFile(path)
}

func mergedReport(security *model.Security, kind string) *model.HoldingReport {
	if security.Report == nil {
		return nil
	}
	if security.AltReport == nil {
		return security.Report
	}
	if len(security.Report.Taxonomies[kind]) > 0 {
		return security.Report
	}
	clone := model.NewHoldingReport()
	clone.SecID = security.Report.SecID
	clone.Security = security.Report.Security
	clone.Type = security.Report.Type
	clone.FundType = security.Report.FundType
	for k, v := range security.Report.Taxonomies {
		clone.Taxonomies[k] = v
	}
	for k, v := range security.AltReport.Taxonomies {
		if len(clone.Taxonomies[k]) == 0 {
			clone.Taxonomies[k] = v
		}
	}
	return clone
}

func ensureAssignment(assignments *Node, ref string, weight int64, rankCounter *int) {
	for _, assignment := range assignments.ChildrenNamed("assignment") {
		vehicle := assignment.Child("investmentVehicle")
		if vehicle != nil && vehicle.Attr("reference") == ref {
			assignment.EnsureChild("weight").Text = strconv.FormatInt(weight, 10)
			return
		}
	}
	assignment := &Node{Name: xml.Name{Local: "assignment"}}
	investment := &Node{Name: xml.Name{Local: "investmentVehicle"}}
	investment.SetAttr("class", "security")
	investment.SetAttr("reference", ref)
	assignment.Append(investment)
	assignment.Append(TextNode("weight", strconv.FormatInt(weight, 10)))
	assignment.Append(TextNode("rank", strconv.Itoa(*rankCounter)))
	*rankCounter++
	assignments.Append(assignment)
}

func ensureClassification(parent *Node, name string, colorIndex *int) *Node {
	for _, class := range parent.ChildrenNamed("classification") {
		if childText(class, "name") == name {
			return class
		}
	}
	class := newClassification(name, taxonomy.Colors[*colorIndex%len(taxonomy.Colors)], len(parent.ChildrenNamed("classification")))
	*colorIndex++
	parent.Append(class)
	return class
}

func newClassification(name, color string, rank int) *Node {
	class := &Node{Name: xml.Name{Local: "classification"}}
	class.Append(TextNode("id", randomUUID()))
	class.Append(TextNode("name", name))
	class.Append(TextNode("color", color))
	parent := &Node{Name: xml.Name{Local: "parent"}}
	parent.SetAttr("reference", "../../..")
	class.Append(parent)
	class.Append(&Node{Name: xml.Name{Local: "children"}})
	class.Append(&Node{Name: xml.Name{Local: "assignments"}})
	class.Append(TextNode("weight", "0"))
	class.Append(TextNode("rank", strconv.Itoa(rank)))
	return class
}

func (f *File) ensureTaxonomy(kind string) *Node {
	taxonomies := f.root.EnsureChild("taxonomies")
	for _, node := range taxonomies.ChildrenNamed("taxonomy") {
		if childText(node, "name") == kind {
			return node
		}
	}

	node := &Node{Name: xml.Name{Local: "taxonomy"}}
	node.Append(TextNode("id", randomUUID()))
	node.Append(TextNode("name", kind))
	dimensions := &Node{Name: xml.Name{Local: "dimensions"}}
	for _, value := range taxonomy.RootDimensions(kind) {
		dimensions.Append(TextNode("string", value))
	}
	node.Append(dimensions)
	root := &Node{Name: xml.Name{Local: "root"}}
	root.Append(TextNode("id", randomUUID()))
	root.Append(TextNode("name", kind))
	root.Append(TextNode("color", taxonomy.Colors[0]))
	root.Append(&Node{Name: xml.Name{Local: "children"}})
	root.Append(&Node{Name: xml.Name{Local: "assignments"}})
	root.Append(TextNode("weight", "10000"))
	root.Append(TextNode("rank", "0"))
	if key, ok := taxonomy.RootDataKey(kind); ok {
		data := &Node{Name: xml.Name{Local: "data"}}
		entry := &Node{Name: xml.Name{Local: "entry"}}
		entry.Append(TextNode("string", "portfolioClassificationKey"))
		entry.Append(TextNode("string", key))
		data.Append(entry)
		root.Append(data)
	}
	node.Append(root)
	taxonomies.Append(node)
	return node
}

func removeZeroAssignments(root *Node) {
	var walk func(*Node)
	walk = func(node *Node) {
		for _, child := range append([]*Node(nil), node.Children...) {
			if child.Name.Local == "assignments" {
				for _, assignment := range append([]*Node(nil), child.ChildrenNamed("assignment")...) {
					if childText(assignment, "weight") == "0" {
						child.RemoveChild(assignment)
					}
				}
			}
			walk(child)
		}
	}
	walk(root)
}

func removeAssignmentsForSecurities(root *Node, securities []*model.Security) {
	refs := map[string]bool{}
	for _, security := range securities {
		if security == nil || security.SourceRef == "" {
			continue
		}
		refs[security.SourceRef] = true
		refs[nestedSecurityRef(security.SourceRef)] = true
	}
	if len(refs) == 0 {
		return
	}

	var walk func(*Node)
	walk = func(node *Node) {
		for _, child := range append([]*Node(nil), node.Children...) {
			if child.Name.Local == "assignments" {
				for _, assignment := range append([]*Node(nil), child.ChildrenNamed("assignment")...) {
					vehicle := assignment.Child("investmentVehicle")
					if vehicle != nil && refs[vehicle.Attr("reference")] {
						child.RemoveChild(assignment)
					}
				}
			}
			walk(child)
		}
	}
	walk(root)
}

func scaling(assignments map[string]float64) float64 {
	scale := 1.0
	for {
		sum := 0.0
		for _, weight := range assignments {
			w := taxonomy.ValidateWeight(weight * scale)
			sum += float64(int64(w*100+0.5)) / 100.0
		}
		if sum > 100 && sum < 100.06 {
			scale *= 0.999999
			continue
		}
		return scale
	}
}

func securityRef(index int) string {
	if index == 0 {
		return "../../../../../../../../securities/security"
	}
	return fmt.Sprintf("../../../../../../../../securities/security[%d]", index+1)
}

func nestedSecurityRef(ref string) string {
	return "../../" + ref
}

func childText(node *Node, name string) string {
	if node == nil {
		return ""
	}
	if child := node.Child(name); child != nil {
		return child.Text
	}
	return ""
}

func randomUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	hexs := hex.EncodeToString(b[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s", hexs[0:8], hexs[8:12], hexs[12:16], hexs[16:20], hexs[20:32])
}
