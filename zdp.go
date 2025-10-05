package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// State mapping between names and directories
var states = map[string]string{
	"draft":        "01-draft",
	"under review": "02-under-review",
	"revised":      "03-revised",
	"accepted":     "04-accepted",
	"active":       "05-active",
	"final":        "06-final",
	"deferred":     "07-deferred",
	"rejected":     "08-rejected",
	"withdrawn":    "09-withdrawn",
	"superseded":   "10-superseded",
}

// Reverse mapping: directory to state name
var dirToState = map[string]string{
	"01-draft":        "Draft",
	"02-under-review": "Under Review",
	"03-revised":      "Revised",
	"04-accepted":     "Accepted",
	"05-active":       "Active",
	"06-final":        "Final",
	"07-deferred":     "Deferred",
	"08-rejected":     "Rejected",
	"09-withdrawn":    "Withdrawn",
	"10-superseded":   "Superseded",
}

// Document metadata structure
type DocMetadata struct {
	Number  string
	Title   string
	State   string
	Updated string
}

// parseYAML extracts YAML frontmatter into a map
func parseYAML(content string) (map[string]string, error) {
	re := regexp.MustCompile(`(?s)^---\n(.*?)\n---\n`)
	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return nil, fmt.Errorf("could not find YAML frontmatter")
	}

	yamlContent := matches[1]
	metadata := make(map[string]string)

	for _, line := range strings.Split(yamlContent, "\n") {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			metadata[key] = value
		}
	}

	return metadata, nil
}

// updateYAML updates the state and updated fields in YAML frontmatter
func updateYAML(content, newState string) (string, error) {
	today := time.Now().Format("2006-01-02")

	// Update state field
	stateRe := regexp.MustCompile(`(?m)^state: .*$`)
	content = stateRe.ReplaceAllString(content, "state: "+newState)

	// Update updated field
	updatedRe := regexp.MustCompile(`(?m)^updated: .*$`)
	content = updatedRe.ReplaceAllString(content, "updated: "+today)

	return content, nil
}

// normalizeState converts input to lowercase with spaces
func normalizeState(input string) string {
	// Convert to lowercase and replace hyphens with spaces
	normalized := strings.ToLower(input)
	normalized = strings.ReplaceAll(normalized, "-", " ")
	return strings.TrimSpace(normalized)
}

// getStateDir returns the directory for a given state name
func getStateDir(stateName string) (string, error) {
	normalized := normalizeState(stateName)
	if dir, ok := states[normalized]; ok {
		return dir, nil
	}
	return "", fmt.Errorf("unsupported state")
}

// getTitleCaseState returns the title case version of a state
func getTitleCaseState(stateName string) string {
	normalized := normalizeState(stateName)
	for key := range states {
		if key == normalized {
			// Convert to title case
			words := strings.Split(key, " ")
			for i, word := range words {
				words[i] = strings.Title(word)
			}
			return strings.Join(words, " ")
		}
	}
	return stateName
}

// getCurrentState reads the state from a document
func getCurrentState(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	metadata, err := parseYAML(string(content))
	if err != nil {
		return "", err
	}

	state, ok := metadata["state"]
	if !ok {
		return "", fmt.Errorf("no 'state' field found in document metadata")
	}

	return state, nil
}

// extractDocMetadata extracts number, title, state, and updated date from a document
func extractDocMetadata(docPath string) (*DocMetadata, error) {
	content, err := os.ReadFile(docPath)
	if err != nil {
		return nil, err
	}

	metadata, err := parseYAML(string(content))
	if err != nil {
		return nil, err
	}

	return &DocMetadata{
		Number:  metadata["number"],
		Title:   metadata["title"],
		State:   metadata["state"],
		Updated: metadata["updated"],
	}, nil
}

// moveDocument moves a file from source to destination using git mv
func moveDocument(srcPath, dstPath string) error {
	// Ensure destination directory exists
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	// Use git mv to preserve history
	cmd := exec.Command("git", "mv", srcPath, dstPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git mv failed: %v\nOutput: %s", err, string(output))
	}

	return nil
}

// getGitAuthor extracts the author from git history
func getGitAuthor(filePath string) string {
	cmd := exec.Command("git", "log", "--format=%an", "--reverse", filePath)
	output, err := cmd.Output()
	if err != nil {
		return "Unknown"
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) > 0 && lines[0] != "" {
		return lines[0]
	}
	return "Unknown"
}

// getGitCreatedDate extracts the creation date from git history
func getGitCreatedDate(filePath string) string {
	cmd := exec.Command("git", "log", "--format=%ai", "--reverse", filePath)
	output, err := cmd.Output()
	if err != nil {
		return time.Now().Format("2006-01-02")
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) > 0 && lines[0] != "" {
		// Extract just the date portion (YYYY-MM-DD)
		parts := strings.Fields(lines[0])
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return time.Now().Format("2006-01-02")
}

// getGitUpdatedDate extracts the last modified date from git history
func getGitUpdatedDate(filePath string) string {
	cmd := exec.Command("git", "log", "--format=%ai", "-1", filePath)
	output, err := cmd.Output()
	if err != nil {
		return time.Now().Format("2006-01-02")
	}

	dateStr := strings.TrimSpace(string(output))
	if dateStr != "" {
		// Extract just the date portion (YYYY-MM-DD)
		parts := strings.Fields(dateStr)
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return time.Now().Format("2006-01-02")
}

// extractNumberFromFilename extracts and pads the number from a filename
func extractNumberFromFilename(filename string) string {
	re := regexp.MustCompile(`^(\d+)-`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) > 1 {
		// Pad to 4 digits
		num := matches[1]
		for len(num) < 4 {
			num = "0" + num
		}
		return num
	}
	return "0000"
}

// extractTitleFromContent finds the first # heading or infers from filename
func extractTitleFromContent(content, filename string) string {
	// Look for first # heading
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(trimmed[2:])
		}
	}

	// Infer from filename
	re := regexp.MustCompile(`^\d+-(.+)\.md$`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) > 1 {
		slug := matches[1]
		// Convert slug to title case
		words := strings.Split(slug, "-")
		for i, word := range words {
			words[i] = strings.Title(word)
		}
		return strings.Join(words, " ")
	}

	return "Untitled Document"
}

// hasYAMLFrontmatter checks if content has YAML frontmatter
func hasYAMLFrontmatter(content string) bool {
	return strings.HasPrefix(strings.TrimSpace(content), "---\n")
}

// buildCompleteYAML constructs a complete YAML frontmatter block
func buildCompleteYAML(metadata map[string]string) string {
	yaml := "---\n"
	yaml += fmt.Sprintf("number: %s\n", metadata["number"])
	yaml += fmt.Sprintf("title: \"%s\"\n", metadata["title"])
	yaml += fmt.Sprintf("author: %s\n", metadata["author"])
	yaml += fmt.Sprintf("created: %s\n", metadata["created"])
	yaml += fmt.Sprintf("updated: %s\n", metadata["updated"])
	yaml += fmt.Sprintf("state: %s\n", metadata["state"])
	yaml += fmt.Sprintf("supersedes: %s\n", metadata["supersedes"])
	yaml += fmt.Sprintf("superseded-by: %s\n", metadata["superseded-by"])
	yaml += "---\n\n"
	return yaml
}

// listAllDocuments returns documents grouped by state
func listAllDocuments() map[string][]string {
	result := make(map[string][]string)

	// Scan all state directories
	for stateName, dir := range states {
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		var docs []string
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".md") {
				docs = append(docs, file.Name())
			}
		}

		if len(docs) > 0 {
			sort.Strings(docs)
			titleCase := getTitleCaseState(stateName)
			result[titleCase] = docs
		}
	}

	return result
}

// addHeadersToDocument adds or completes YAML frontmatter for a document
func addHeadersToDocument(docPath string) {
	// Validate file exists
	if _, err := os.Stat(docPath); os.IsNotExist(err) {
		panic(fmt.Sprintf("Error: File not found: %s", docPath))
	}

	// Read the file
	content, err := os.ReadFile(docPath)
	if err != nil {
		panic(fmt.Sprintf("Error: Failed to read file: %v", err))
	}

	contentStr := string(content)
	filename := filepath.Base(docPath)

	// Extract metadata
	number := extractNumberFromFilename(filename)
	title := extractTitleFromContent(contentStr, filename)
	author := getGitAuthor(docPath)
	created := getGitCreatedDate(docPath)
	updated := getGitUpdatedDate(docPath)

	// Build metadata map with defaults
	metadata := map[string]string{
		"number":        number,
		"title":         title,
		"author":        author,
		"created":       created,
		"updated":       updated,
		"state":         "Draft",
		"supersedes":    "None",
		"superseded-by": "None",
	}

	var newContent string
	var addedFields []string

	if hasYAMLFrontmatter(contentStr) {
		// Parse existing YAML and merge with discovered metadata
		existing, err := parseYAML(contentStr)
		if err != nil {
			panic(fmt.Sprintf("Error: Failed to parse existing YAML: %v", err))
		}

		// Merge: existing values take precedence over discovered ones
		for key, value := range existing {
			if value != "" {
				metadata[key] = value
			}
		}

		// Track which fields were added (not in existing)
		requiredFields := []string{"number", "title", "author", "created", "updated", "state", "supersedes", "superseded-by"}
		for _, field := range requiredFields {
			if _, exists := existing[field]; !exists || existing[field] == "" {
				addedFields = append(addedFields, field)
			}
		}

		// Remove old frontmatter and rebuild
		re := regexp.MustCompile(`(?s)^---\n.*?\n---\n\n?`)
		bodyContent := re.ReplaceAllString(contentStr, "")
		newContent = buildCompleteYAML(metadata) + bodyContent
	} else {
		// No frontmatter exists, add it
		addedFields = []string{"number", "title", "author", "created", "updated", "state", "supersedes", "superseded-by"}
		newContent = buildCompleteYAML(metadata) + contentStr
	}

	// Write updated content
	if err := os.WriteFile(docPath, []byte(newContent), 0644); err != nil {
		panic(fmt.Sprintf("Error: Failed to write file: %v", err))
	}

	// Report what was done
	if len(addedFields) > 0 {
		fmt.Printf("Added/updated headers in %s:\n", filename)
		for _, field := range addedFields {
			fmt.Printf("  %s: %s\n", field, metadata[field])
		}
	} else {
		fmt.Printf("All headers already present in %s\n", filename)
	}
}

// updateIndex updates the 00-index.md file when a document changes state
func updateIndex(docPath, oldState, newState string) error {
	indexPath := "00-index.md"
	content, err := os.ReadFile(indexPath)
	if err != nil {
		return err
	}

	// Extract document metadata
	meta, err := extractDocMetadata(docPath)
	if err != nil {
		return err
	}

	indexContent := string(content)
	today := time.Now().Format("2006-01-02")

	// Update the table row
	indexContent = updateIndexTable(indexContent, meta.Number, newState, today)

	// Update state sections
	oldDir, _ := getStateDir(oldState)
	newDir, _ := getStateDir(newState)

	oldPath := filepath.Join(oldDir, filepath.Base(docPath))
	newPath := filepath.Join(newDir, filepath.Base(docPath))

	indexContent = removeFromStateSection(indexContent, oldPath, oldState)
	indexContent = addToStateSection(indexContent, newPath, newState, meta.Title, meta.Number)

	// Write updated index
	return os.WriteFile(indexPath, []byte(indexContent), 0644)
}

// updateIndexTable updates a row in the "All Documents by Number" table
func updateIndexTable(content, docNumber, newState, newUpdated string) string {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		if strings.HasPrefix(line, "| "+docNumber+" |") {
			// Update this row
			parts := strings.Split(line, "|")
			if len(parts) >= 5 {
				parts[3] = " " + newState + " "
				parts[4] = " " + newUpdated + " "
				line = strings.Join(parts, "|")
			}
		}
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// removeFromStateSection removes a document from its old state section
func removeFromStateSection(content, docPath, state string) string {
	lines := strings.Split(content, "\n")
	var result []string

	inStateSection := false
	stateHeader := "### " + state

	for lineIdx, line := range lines {
		// Check if we're entering the state section
		if line == stateHeader {
			inStateSection = true
			result = append(result, line)
			continue
		}

		// Check if we're leaving the state section
		if inStateSection && (strings.HasPrefix(line, "### ") || strings.HasPrefix(line, "## ")) {
			inStateSection = false
		}

		// Skip the line if it matches our document
		if inStateSection && strings.Contains(line, "]("+docPath+")") {
			// Check if this was the only document in the section
			// If so, also remove the section header
			if lineIdx > 0 && strings.HasPrefix(result[len(result)-1], stateHeader) {
				// Check if next line is also a section header
				if lineIdx+1 < len(lines) && (strings.HasPrefix(lines[lineIdx+1], "### ") || strings.HasPrefix(lines[lineIdx+1], "## ")) {
					// Remove the section header
					result = result[:len(result)-1]
				}
			}
			continue
		}

		result = append(result, line)
	}

	// Clean up empty sections - look ahead to find sections with no content
	var cleaned []string
	skipUntilIdx := -1

	for idx, line := range result {
		if idx <= skipUntilIdx {
			continue
		}

		if strings.HasPrefix(line, "### ") {
			// Look ahead to find if this section has any content
			hasContent := false
			for j := idx + 1; j < len(result); j++ {
				nextLine := result[j]
				// If we hit another section, this section is empty
				if strings.HasPrefix(nextLine, "### ") || strings.HasPrefix(nextLine, "## ") {
					skipUntilIdx = j - 1 // Skip to just before next section
					break
				}
				// If we find content (not blank line), section is not empty
				if nextLine != "" && !strings.HasPrefix(nextLine, "### ") && !strings.HasPrefix(nextLine, "## ") {
					hasContent = true
					break
				}
			}
			// Skip this section header if it has no content
			if !hasContent {
				continue
			}
		}

		cleaned = append(cleaned, line)
	}

	return strings.Join(cleaned, "\n")
}

// addToStateSection adds a document to its new state section
func addToStateSection(content, docPath, state, title, number string) string {
	lines := strings.Split(content, "\n")
	var result []string

	stateHeader := "### " + state
	fullStateHeader := "### " + state

	inStateSection := false
	sectionExists := false
	inserted := false
	docNum, _ := strconv.Atoi(number)

	for _, line := range lines {
		// Check if we're at the state section
		if line == stateHeader {
			sectionExists = true
			inStateSection = true
			result = append(result, line)
			continue
		}

		// Check if we're leaving the state section
		if inStateSection && (strings.HasPrefix(line, "### ") || strings.HasPrefix(line, "## ")) {
			// Insert before leaving if not yet inserted
			if !inserted {
				newLine := fmt.Sprintf("- [%s - %s](%s)", number, title, docPath)
				result = append(result, newLine)
				inserted = true
			}
			inStateSection = false
		}

		// Insert in sorted position within the section
		if inStateSection && strings.HasPrefix(line, "- [") && !inserted {
			// Extract number from this line
			re := regexp.MustCompile(`^\- \[(\d+)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				existingNum, _ := strconv.Atoi(matches[1])
				if docNum < existingNum {
					newLine := fmt.Sprintf("- [%s - %s](%s)", number, title, docPath)
					result = append(result, newLine)
					inserted = true
				}
			}
		}

		result = append(result, line)
	}

	// If section doesn't exist, create it
	if !sectionExists {
		// Find where to insert the new section (after "## Documents by State")
		for lineNum, line := range result {
			if line == "## Documents by State" {
				// Insert new section
				newSection := []string{
					"",
					fullStateHeader,
					fmt.Sprintf("- [%s - %s](%s)", number, title, docPath),
				}
				result = append(result[:lineNum+1], append(newSection, result[lineNum+1:]...)...)
				inserted = true
				break
			}
		}
	}

	// If still not inserted and we were in the section, add at end
	if !inserted && inStateSection {
		newLine := fmt.Sprintf("- [%s - %s](%s)", number, title, docPath)
		result = append(result, newLine)
	}

	return strings.Join(result, "\n")
}

// addToIndex adds a document to the index if not already present
func addToIndex(docPath string) error {
	indexPath := "00-index.md"
	content, err := os.ReadFile(indexPath)
	if err != nil {
		return err
	}

	// Extract document metadata
	meta, err := extractDocMetadata(docPath)
	if err != nil {
		return err
	}

	indexContent := string(content)

	// Check if document is in table
	tableHasDoc := strings.Contains(indexContent, "| "+meta.Number+" |")

	// Check if document is in state section
	stateSectionHasDoc := strings.Contains(indexContent, "]("+docPath+")")

	if tableHasDoc && stateSectionHasDoc {
		fmt.Println("Document already indexed correctly")
		return nil
	}

	// Add to table if missing
	if !tableHasDoc {
		indexContent = addToIndexTable(indexContent, meta)
	}

	// Add to state section if missing
	if !stateSectionHasDoc {
		indexContent = addToStateSection(indexContent, docPath, meta.State, meta.Title, meta.Number)
	}

	// Write updated index
	if err := os.WriteFile(indexPath, []byte(indexContent), 0644); err != nil {
		return err
	}

	fmt.Printf("Added %s to index\n", filepath.Base(docPath))
	return nil
}

// addToIndexTable adds a row to the "All Documents by Number" table
func addToIndexTable(content string, meta *DocMetadata) string {
	lines := strings.Split(content, "\n")
	var result []string

	docNum, _ := strconv.Atoi(meta.Number)
	inserted := false

	for i, line := range lines {
		// Find the table section
		if strings.HasPrefix(line, "| ") && strings.Contains(line, " | ") {
			// Check if this is a data row (not header)
			if !strings.Contains(line, "---|") {
				// Extract number from this row
				parts := strings.Split(line, "|")
				if len(parts) >= 2 {
					rowNumStr := strings.TrimSpace(parts[1])
					rowNum, err := strconv.Atoi(rowNumStr)
					if err == nil && docNum < rowNum && !inserted {
						// Insert before this row
						newRow := fmt.Sprintf("| %s | %s | %s | %s |", meta.Number, meta.Title, meta.State, meta.Updated)
						result = append(result, newRow)
						inserted = true
					}
				}
			}
		}

		result = append(result, line)

		// If we just passed the table header separator, check if table is empty
		if strings.Contains(line, "---|") && i+1 < len(lines) {
			nextLine := lines[i+1]
			if !strings.HasPrefix(nextLine, "| ") && !inserted {
				// Table is empty or we're at the end, insert here
				newRow := fmt.Sprintf("| %s | %s | %s | %s |", meta.Number, meta.Title, meta.State, meta.Updated)
				result = append(result, newRow)
				inserted = true
			}
		}
	}

	return strings.Join(result, "\n")
}

// transitionDocument transitions a document to a new state
func transitionDocument(docPath, newState string) {
	// Validate file exists
	if _, err := os.Stat(docPath); os.IsNotExist(err) {
		panic(fmt.Sprintf("Error: File not found: %s", docPath))
	}

	// Check if document has headers, add them if missing
	content, _ := os.ReadFile(docPath)
	if !hasYAMLFrontmatter(string(content)) {
		fmt.Println("Document missing headers, adding them automatically...")
		addHeadersToDocument(docPath)
	}

	// Get current state
	currentState, err := getCurrentState(docPath)
	if err != nil {
		panic(fmt.Sprintf("Error: Could not parse YAML frontmatter in %s", docPath))
	}

	// Normalize and validate new state
	normalized := normalizeState(newState)
	newStateDir, err := getStateDir(newState)
	if err != nil {
		// List supported states
		var supported []string
		for state := range states {
			supported = append(supported, getTitleCaseState(state))
		}
		sort.Strings(supported)
		panic(fmt.Sprintf("Error: Unsupported state \"%s\". Supported states are:\n%s", newState, strings.Join(supported, ", ")))
	}

	// Check if already in that state
	if normalizeState(currentState) == normalized {
		panic(fmt.Sprintf("Error: Document is already in state \"%s\"", currentState))
	}

	// Read and update document
	content, _ = os.ReadFile(docPath)
	newStateTitleCase := getTitleCaseState(newState)
	updatedContent, err := updateYAML(string(content), newStateTitleCase)
	if err != nil {
		panic(fmt.Sprintf("Error: Failed to update YAML: %v", err))
	}

	// Write updated content back to the same file first
	if err := os.WriteFile(docPath, []byte(updatedContent), 0644); err != nil {
		panic(fmt.Sprintf("Error: Failed to update file: %v", err))
	}

	// Now use git mv to move to new location
	filename := filepath.Base(docPath)
	newPath := filepath.Join(newStateDir, filename)

	if err := moveDocument(docPath, newPath); err != nil {
		panic(fmt.Sprintf("Error: Failed to move document: %v", err))
	}

	// Update index
	if err := updateIndex(newPath, currentState, newStateTitleCase); err != nil {
		panic(fmt.Sprintf("Error: Failed to update index: %v", err))
	}

	fmt.Printf("Moved %s from %s to %s\n", filename, currentState, newStateTitleCase)
	fmt.Println("Updated index")
}

// moveToMatchHeader moves a document to the directory matching its header state
func moveToMatchHeader(docPath string) {
	// Validate file exists
	if _, err := os.Stat(docPath); os.IsNotExist(err) {
		panic(fmt.Sprintf("Error: File not found: %s", docPath))
	}

	// Check if document has headers, add them if missing
	content, _ := os.ReadFile(docPath)
	if !hasYAMLFrontmatter(string(content)) {
		fmt.Println("Document missing headers, adding them automatically...")
		addHeadersToDocument(docPath)
	}

	// Get state from header
	headerState, err := getCurrentState(docPath)
	if err != nil {
		panic(fmt.Sprintf("Error: Could not parse YAML frontmatter in %s", docPath))
	}

	// Get directory for that state
	stateDir, err := getStateDir(headerState)
	if err != nil {
		var supported []string
		for state := range states {
			supported = append(supported, getTitleCaseState(state))
		}
		sort.Strings(supported)
		panic(fmt.Sprintf("Error: Unsupported state \"%s\". Supported states are:\n%s", headerState, strings.Join(supported, ", ")))
	}

	// Check if already in correct directory
	currentDir := filepath.Dir(docPath)
	if currentDir == stateDir {
		panic(fmt.Sprintf("Error: Document is already in the correct directory for state \"%s\"", headerState))
	}

	// Move the file
	filename := filepath.Base(docPath)
	newPath := filepath.Join(stateDir, filename)

	if err := moveDocument(docPath, newPath); err != nil {
		panic(fmt.Sprintf("Error: Failed to move document: %v", err))
	}

	fmt.Printf("Moved %s to %s (state: %s)\n", filename, stateDir, headerState)
}

// listStates lists all supported states
func listStates() {
	var stateNames []string
	for state := range states {
		stateNames = append(stateNames, getTitleCaseState(state))
	}
	sort.Strings(stateNames)

	for _, state := range stateNames {
		fmt.Println(state)
	}
}

// listDocuments lists all documents by state
func listDocuments() {
	docs := listAllDocuments()

	// Get sorted state names
	var stateNames []string
	for state := range docs {
		stateNames = append(stateNames, state)
	}
	sort.Strings(stateNames)

	for _, state := range stateNames {
		fmt.Println(state)
		for _, doc := range docs[state] {
			fmt.Printf(" - %s\n", doc)
		}
		fmt.Println()
	}
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		// Mode 3: List all documents by state
		listDocuments()
		return
	}

	if len(args) == 1 {
		if args[0] == "states" {
			// Mode 4: List supported states
			listStates()
			return
		}

		// Mode 2: Move to directory matching header state
		moveToMatchHeader(args[0])
		return
	}

	if len(args) == 2 {
		if args[0] == "index" {
			// Mode 5: Add document to index
			if err := addToIndex(args[1]); err != nil {
				panic(fmt.Sprintf("Error: %v", err))
			}
			return
		}

		if args[0] == "add-headers" {
			// Mode 6: Add or update YAML frontmatter headers
			addHeadersToDocument(args[1])
			return
		}

		// Mode 1: Transition to new state
		transitionDocument(args[0], args[1])
		return
	}

	fmt.Println("Usage:")
	fmt.Println("  zdp.go                           - List all documents by state")
	fmt.Println("  zdp.go states                    - List supported states")
	fmt.Println("  zdp.go <doc.md> <new-state>      - Transition document to new state")
	fmt.Println("  zdp.go <doc.md>                  - Move document to match header state")
	fmt.Println("  zdp.go index <doc.md>            - Add document to index")
	fmt.Println("  zdp.go add-headers <doc.md>      - Add/update YAML frontmatter headers")
}
