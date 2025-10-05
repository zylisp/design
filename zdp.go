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

// cleanupSectionFormatting ensures proper spacing around headings and within bullet lists
func cleanupSectionFormatting(lines []string) []string {
	var result []string

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Check if this is a section header (### or ##)
		isHeader := strings.HasPrefix(line, "### ") || strings.HasPrefix(line, "## ")

		if isHeader {
			// Ensure exactly one blank line before the header
			// Remove any trailing blank lines from result
			for len(result) > 0 && result[len(result)-1] == "" {
				result = result[:len(result)-1]
			}
			// Add exactly one blank line (unless this is the very first line)
			if len(result) > 0 {
				result = append(result, "")
			}

			// Add the header
			result = append(result, line)

			// Ensure exactly one blank line after the header
			// Skip any blank lines that follow
			j := i + 1
			for j < len(lines) && lines[j] == "" {
				j++
			}
			// Add exactly one blank line (unless we're at the end or next is another header)
			if j < len(lines) && !strings.HasPrefix(lines[j], "### ") && !strings.HasPrefix(lines[j], "## ") {
				result = append(result, "")
			}
			i = j - 1 // Skip the blank lines we just processed
			continue
		}

		// Check if this is a bullet item
		isBullet := strings.HasPrefix(line, "- [")

		if isBullet {
			// Add the bullet
			result = append(result, line)

			// Look ahead: if next line is also a bullet, skip any blank lines between them
			if i+1 < len(lines) {
				j := i + 1
				// Skip blank lines
				for j < len(lines) && lines[j] == "" {
					j++
				}
				// If the next non-blank line is also a bullet, skip the blanks
				if j < len(lines) && strings.HasPrefix(lines[j], "- [") {
					i = j - 1 // Skip blank lines between bullets
					continue
				}
			}
			continue
		}

		// For non-header, non-bullet lines, just add them
		result = append(result, line)
	}

	return result
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

	// Apply formatting cleanup
	cleaned = cleanupSectionFormatting(cleaned)

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

	// Apply formatting cleanup
	result = cleanupSectionFormatting(result)

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

// getHighestDocNumber returns the highest document number from the index
func getHighestDocNumber() (int, error) {
	indexPath := "00-index.md"
	content, err := os.ReadFile(indexPath)
	if err != nil {
		return 0, err
	}

	entries := parseIndexTableEntries(string(content))
	highest := 0

	for numStr := range entries {
		num, err := strconv.Atoi(numStr)
		if err == nil && num > highest {
			highest = num
		}
	}

	return highest, nil
}

// hasNumberPrefix checks if a filename starts with a number prefix
func hasNumberPrefix(filename string) bool {
	re := regexp.MustCompile(`^\d{4}-`)
	return re.MatchString(filename)
}

// renameWithNumber renames a file to include a number prefix
func renameWithNumber(filePath string, number int) (string, error) {
	dir := filepath.Dir(filePath)
	filename := filepath.Base(filePath)

	// Format number with leading zeros (4 digits)
	paddedNum := fmt.Sprintf("%04d", number)

	// Create new filename
	newFilename := fmt.Sprintf("%s-%s", paddedNum, filename)
	newPath := filepath.Join(dir, newFilename)

	// Rename the file
	if err := os.Rename(filePath, newPath); err != nil {
		return "", err
	}

	return newPath, nil
}

// isInProjectDir checks if a file is within the design project directory
func isInProjectDir(filePath string) (bool, error) {
	// Get absolute path of the file
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return false, err
	}

	// Get current working directory (should be the project dir)
	cwd, err := os.Getwd()
	if err != nil {
		return false, err
	}

	// Check if file path starts with project directory
	return strings.HasPrefix(absFilePath, cwd), nil
}

// isInStateDir checks if a file is in one of the state directories
func isInStateDir(filePath string) bool {
	dir := filepath.Dir(filePath)
	dirName := filepath.Base(dir)

	// Check if the directory name matches any state directory
	for _, stateDir := range states {
		if dirName == stateDir {
			return true
		}
	}

	return false
}

// addToIndexTable adds a row to the "All Documents by Number" table
func addToIndexTable(content string, meta *DocMetadata) string {
	lines := strings.Split(content, "\n")
	var result []string

	docNum, _ := strconv.Atoi(meta.Number)
	inserted := false
	inTable := false
	passedSeparator := false
	lastDataRowIdx := -1

	for i, line := range lines {
		// Detect table start
		if strings.HasPrefix(line, "| Number | Title") {
			inTable = true
		}

		// Detect header separator
		if inTable && strings.Contains(line, "---|") {
			passedSeparator = true
		}

		// Track data rows (after separator, starting with "| " and containing numbers)
		if inTable && passedSeparator && strings.HasPrefix(line, "| ") {
			parts := strings.Split(line, "|")
			if len(parts) >= 2 {
				rowNumStr := strings.TrimSpace(parts[1])
				_, err := strconv.Atoi(rowNumStr)
				if err == nil {
					lastDataRowIdx = i
				}
			}
		}

		// If we're past the separator and in a data row, check if we should insert before it
		if inTable && passedSeparator && strings.HasPrefix(line, "| ") && !inserted {
			parts := strings.Split(line, "|")
			if len(parts) >= 2 {
				rowNumStr := strings.TrimSpace(parts[1])
				rowNum, err := strconv.Atoi(rowNumStr)
				if err == nil && docNum < rowNum {
					// Insert before this row
					newRow := fmt.Sprintf("| %s | %s | %s | %s |", meta.Number, meta.Title, meta.State, meta.Updated)
					result = append(result, newRow)
					inserted = true
				}
			}
		}

		result = append(result, line)

		// If we just left the table and haven't inserted, append at the end
		if inTable && !strings.HasPrefix(line, "|") && lastDataRowIdx >= 0 && !inserted {
			// Insert before this line (after the last data row)
			newRow := fmt.Sprintf("| %s | %s | %s | %s |", meta.Number, meta.Title, meta.State, meta.Updated)
			result = result[:len(result)-1]  // Remove current line
			result = append(result, newRow)  // Add new row
			result = append(result, line)    // Add back current line
			inserted = true
			inTable = false
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

// IndexEntry represents an entry in the index table
type IndexEntry struct {
	Number  string
	Title   string
	State   string
	Updated string
}

// getGitTrackedDocs returns all git-tracked .md files in state directories
func getGitTrackedDocs() []string {
	var allDocs []string

	// Get git-tracked files for each state directory
	for _, dir := range states {
		cmd := exec.Command("git", "ls-files", dir+"/*.md")
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		files := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, file := range files {
			if file != "" {
				allDocs = append(allDocs, file)
			}
		}
	}

	return allDocs
}

// parseIndexTableEntries parses the "All Documents by Number" table
func parseIndexTableEntries(content string) map[string]IndexEntry {
	entries := make(map[string]IndexEntry)
	lines := strings.Split(content, "\n")

	inTable := false
	for _, line := range lines {
		// Start of table
		if strings.HasPrefix(line, "| Number | Title") {
			inTable = true
			continue
		}

		// Table separator line
		if inTable && strings.Contains(line, "---|") {
			continue
		}

		// End of table
		if inTable && !strings.HasPrefix(line, "|") {
			break
		}

		// Parse table row
		if inTable && strings.HasPrefix(line, "|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 5 {
				number := strings.TrimSpace(parts[1])
				title := strings.TrimSpace(parts[2])
				state := strings.TrimSpace(parts[3])
				updated := strings.TrimSpace(parts[4])

				if number != "" && number != "Number" {
					entries[number] = IndexEntry{
						Number:  number,
						Title:   title,
						State:   state,
						Updated: updated,
					}
				}
			}
		}
	}

	return entries
}

// getFilesInStateSection extracts document paths from a state section
func getFilesInStateSection(content, state string) []string {
	var files []string
	lines := strings.Split(content, "\n")

	stateHeader := "### " + state
	inSection := false

	for _, line := range lines {
		if line == stateHeader {
			inSection = true
			continue
		}

		if inSection && (strings.HasPrefix(line, "### ") || strings.HasPrefix(line, "## ")) {
			break
		}

		if inSection && strings.HasPrefix(line, "- [") {
			// Extract path from markdown link: - [0001 - Title](path/to/file.md)
			re := regexp.MustCompile(`\]\(([^)]+)\)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				files = append(files, matches[1])
			}
		}
	}

	return files
}

// syncIndexTable synchronizes the table with git-tracked documents
func syncIndexTable(indexContent string, gitDocs []string) (string, []string) {
	var changes []string
	currentEntries := parseIndexTableEntries(indexContent)

	// Process each git-tracked document
	for _, docPath := range gitDocs {
		meta, err := extractDocMetadata(docPath)
		if err != nil {
			changes = append(changes, fmt.Sprintf("  ⚠ Skipped %s: %v", filepath.Base(docPath), err))
			continue
		}

		existing, exists := currentEntries[meta.Number]

		if !exists {
			// Add new entry to table
			indexContent = addToIndexTable(indexContent, meta)
			changes = append(changes, fmt.Sprintf("  ✓ Added: %s", filepath.Base(docPath)))
		} else {
			// Check if updated date differs
			if existing.Updated != meta.Updated {
				indexContent = updateIndexTable(indexContent, meta.Number, meta.State, meta.Updated)
				changes = append(changes, fmt.Sprintf("  ✓ Updated date: %s (%s → %s)", filepath.Base(docPath), existing.Updated, meta.Updated))
			}
			// Check if state differs
			if existing.State != meta.State {
				indexContent = updateIndexTable(indexContent, meta.Number, meta.State, meta.Updated)
				changes = append(changes, fmt.Sprintf("  ✓ Updated state: %s (%s → %s)", filepath.Base(docPath), existing.State, meta.State))
			}
		}
	}

	return indexContent, changes
}

// syncStateSection synchronizes a state section with its directory
func syncStateSection(indexContent, state, stateDir string) (string, []string) {
	var changes []string

	// Get files in directory
	dirFiles, err := os.ReadDir(stateDir)
	if err != nil {
		return indexContent, changes
	}

	var dirDocs []string
	for _, file := range dirFiles {
		if strings.HasSuffix(file.Name(), ".md") {
			dirDocs = append(dirDocs, filepath.Join(stateDir, file.Name()))
		}
	}

	// Get files in section
	sectionFiles := getFilesInStateSection(indexContent, state)

	// Find files in directory but not in section (need to add)
	sectionFileSet := make(map[string]bool)
	for _, f := range sectionFiles {
		sectionFileSet[f] = true
	}

	for _, docPath := range dirDocs {
		if !sectionFileSet[docPath] {
			// Extract metadata and add to section
			meta, err := extractDocMetadata(docPath)
			if err != nil {
				changes = append(changes, fmt.Sprintf("  ⚠ Skipped %s: %v", filepath.Base(docPath), err))
				continue
			}
			indexContent = addToStateSection(indexContent, docPath, state, meta.Title, meta.Number)
			changes = append(changes, fmt.Sprintf("  ✓ Added: %s", filepath.Base(docPath)))
		}
	}

	// Find files in section but not in directory (need to remove)
	dirFileSet := make(map[string]bool)
	for _, f := range dirDocs {
		dirFileSet[f] = true
	}

	for _, docPath := range sectionFiles {
		if !dirFileSet[docPath] {
			indexContent = removeFromStateSection(indexContent, docPath, state)
			changes = append(changes, fmt.Sprintf("  ✗ Removed: %s (file not found)", filepath.Base(docPath)))
		}
	}

	return indexContent, changes
}

// addDocument adds a new document to the repository with full processing
func addDocument(docPath string) {
	fmt.Printf("Adding document: %s\n\n", docPath)

	// Validate file exists
	if _, err := os.Stat(docPath); os.IsNotExist(err) {
		panic(fmt.Sprintf("Error: File not found: %s", docPath))
	}

	// Step 1: Number Assignment (FIRST priority)
	filename := filepath.Base(docPath)
	if !hasNumberPrefix(filename) {
		fmt.Println("File does not have a numbered prefix, assigning number...")

		// Get highest number from index
		highest, err := getHighestDocNumber()
		if err != nil {
			panic(fmt.Sprintf("Error: Failed to read index: %v", err))
		}

		nextNum := highest + 1
		fmt.Printf("Assigning number: %04d\n", nextNum)

		// Rename file with number
		newPath, err := renameWithNumber(docPath, nextNum)
		if err != nil {
			panic(fmt.Sprintf("Error: Failed to rename file: %v", err))
		}

		docPath = newPath
		filename = filepath.Base(docPath)
		fmt.Printf("Renamed to: %s\n\n", filename)
	}

	// Step 2: Move to Project Directory
	inProject, err := isInProjectDir(docPath)
	if err != nil {
		panic(fmt.Sprintf("Error: Failed to check project directory: %v", err))
	}

	if !inProject {
		fmt.Println("File is outside project directory, moving to project root...")

		cwd, _ := os.Getwd()
		newPath := filepath.Join(cwd, filename)

		if err := os.Rename(docPath, newPath); err != nil {
			panic(fmt.Sprintf("Error: Failed to move file to project: %v", err))
		}

		docPath = newPath
		fmt.Printf("Moved to: %s\n\n", docPath)
	}

	// Step 3: State Directory Placement
	if !isInStateDir(docPath) {
		fmt.Println("File is not in a state directory, moving to draft (01-draft)...")

		draftDir := "01-draft"
		newPath := filepath.Join(draftDir, filename)

		// Ensure draft directory exists
		if err := os.MkdirAll(draftDir, 0755); err != nil {
			panic(fmt.Sprintf("Error: Failed to create draft directory: %v", err))
		}

		if err := os.Rename(docPath, newPath); err != nil {
			panic(fmt.Sprintf("Error: Failed to move file to draft: %v", err))
		}

		docPath = newPath
		fmt.Printf("Moved to: %s\n\n", docPath)
	}

	// Step 4: Add YAML Frontmatter Headers
	content, _ := os.ReadFile(docPath)
	if !hasYAMLFrontmatter(string(content)) || strings.Contains(string(content), "number: NNNN") {
		fmt.Println("Adding/updating YAML frontmatter headers...")
		addHeadersToDocument(docPath)
		fmt.Println()
	}

	// Step 5: Sync State Header with Directory
	// Get directory-based state
	dir := filepath.Dir(docPath)
	dirName := filepath.Base(dir)
	dirState, exists := dirToState[dirName]

	if exists {
		// Check current state in document
		currentState, err := getCurrentState(docPath)
		if err == nil && normalizeState(currentState) != normalizeState(dirState) {
			fmt.Printf("State header mismatch, updating to match directory: %s\n", dirState)

			content, _ := os.ReadFile(docPath)
			updatedContent, err := updateYAML(string(content), dirState)
			if err != nil {
				panic(fmt.Sprintf("Error: Failed to update YAML: %v", err))
			}

			if err := os.WriteFile(docPath, []byte(updatedContent), 0644); err != nil {
				panic(fmt.Sprintf("Error: Failed to write file: %v", err))
			}
			fmt.Println()
		}
	}

	// Step 6: Git Add
	fmt.Println("Adding file to git...")
	cmd := exec.Command("git", "add", docPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		panic(fmt.Sprintf("Error: git add failed: %v\nOutput: %s", err, string(output)))
	}
	fmt.Printf("Git staged: %s\n\n", docPath)

	// Step 7: Update Index
	fmt.Println("Updating index...")
	if err := addToIndex(docPath); err != nil {
		panic(fmt.Sprintf("Error: Failed to update index: %v", err))
	}

	fmt.Printf("\nSuccessfully added document: %s\n", filename)
}

// updateIndexCommand synchronizes the index with git-tracked documents
func updateIndexCommand() {
	fmt.Println("Synchronizing index with git-tracked documents...")
	fmt.Println()

	// Get all git-tracked docs
	gitDocs := getGitTrackedDocs()

	// Read current index
	indexPath := "00-index.md"
	content, err := os.ReadFile(indexPath)
	if err != nil {
		panic(fmt.Sprintf("Error: Failed to read index: %v", err))
	}

	indexContent := string(content)

	// Sync the table
	var allChanges []string
	indexContent, tableChanges := syncIndexTable(indexContent, gitDocs)
	if len(tableChanges) > 0 {
		fmt.Println("Table Updates:")
		for _, change := range tableChanges {
			fmt.Println(change)
		}
		fmt.Println()
		allChanges = append(allChanges, tableChanges...)
	}

	// Sync each state section
	for stateName, stateDir := range states {
		titleCaseState := getTitleCaseState(stateName)
		newContent, sectionChanges := syncStateSection(indexContent, titleCaseState, stateDir)
		indexContent = newContent

		if len(sectionChanges) > 0 {
			fmt.Printf("Section Updates (%s):\n", titleCaseState)
			for _, change := range sectionChanges {
				fmt.Println(change)
			}
			fmt.Println()
			allChanges = append(allChanges, sectionChanges...)
		}
	}

	// Store original content for comparison
	originalContent := indexContent

	// Always run formatting cleanup
	lines := strings.Split(indexContent, "\n")
	cleanedLines := cleanupSectionFormatting(lines)
	indexContent = strings.Join(cleanedLines, "\n")

	// Check if cleanup made formatting changes
	formattingChanged := originalContent != indexContent

	// Report on changes
	if len(allChanges) == 0 && !formattingChanged {
		fmt.Println("Index is already up to date!")
	}

	if formattingChanged {
		fmt.Println("Formatting Cleanup:")
		fmt.Println("  ✓ Fixed section heading spacing and bullet list formatting")
		fmt.Println()
	}

	// Write updated index if there were any changes
	if len(allChanges) > 0 || formattingChanged {
		if err := os.WriteFile(indexPath, []byte(indexContent), 0644); err != nil {
			panic(fmt.Sprintf("Error: Failed to write index: %v", err))
		}

		if len(allChanges) > 0 {
			fmt.Printf("Summary: %d content changes made to index\n", len(allChanges))
		}
		if formattingChanged && len(allChanges) == 0 {
			fmt.Println("Summary: Formatting cleanup applied to index")
		}
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

		if args[0] == "update-index" {
			// Mode 7: Synchronize index with git-tracked documents
			updateIndexCommand()
			return
		}

		// Mode 2: Move to directory matching header state
		moveToMatchHeader(args[0])
		return
	}

	if len(args) == 2 {
		if args[0] == "add" {
			// Mode 8: Add new document with full processing
			addDocument(args[1])
			return
		}

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
	fmt.Println("  zdp.go update-index              - Sync index with git-tracked docs")
	fmt.Println("  zdp.go add <doc.md>              - Add new document with full processing")
	fmt.Println("  zdp.go <doc.md> <new-state>      - Transition document to new state")
	fmt.Println("  zdp.go <doc.md>                  - Move document to match header state")
	fmt.Println("  zdp.go index <doc.md>            - Add document to index")
	fmt.Println("  zdp.go add-headers <doc.md>      - Add/update YAML frontmatter headers")
}
