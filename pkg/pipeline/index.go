// Package pipeline provides a pipeline framework for processing Trust Status Lists (TSLs).
package pipeline

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// TSLIndexEntry represents a single Trust Service List entry in the index
type TSLIndexEntry struct {
	Filename     string // Name of the HTML file
	Title        string // Title of the TSL (usually country name)
	SchemeType   string // Type of the TSL scheme
	Territory    string // Territory code
	Sequence     string // Sequence number
	IssueDate    string // Issue date of the TSL
	NextUpdate   string // Next update date
	URL          string // Link to the HTML file
	TrustService int    // Number of trust services in the TSL
}

// GenerateIndex creates an index.html file in the specified directory.
// The index page lists all TSL HTML files in the directory with metadata and links.
// The index uses PicoCSS for styling to match the TSL HTML files.
//
// Arguments:
//   - arg[0]: Directory path containing TSL HTML files
//   - arg[1]: (Optional) Title for the index page (default: "Trust Service Lists Index")
//
// Example usage in pipeline YAML:
//
//   - generate_index:
//   - /path/to/output/directory
//   - "EU Trust Lists - Index"
func GenerateIndex(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if len(args) < 1 {
		return ctx, fmt.Errorf("missing required directory path argument")
	}

	// Parse arguments
	dirPath := args[0]
	title := "Trust Service Lists Index"
	if len(args) >= 2 {
		title = args[1]
	}

	// Check if the directory exists
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ctx, fmt.Errorf("directory %s does not exist", dirPath)
		}
		return ctx, fmt.Errorf("error accessing directory %s: %w", dirPath, err)
	}
	if !info.IsDir() {
		return ctx, fmt.Errorf("%s is not a directory", dirPath)
	}

	// Find all HTML files in the directory
	entries, err := findTSLHtmlFiles(dirPath)
	if err != nil {
		return ctx, fmt.Errorf("failed to read directory: %w", err)
	}

	if len(entries) == 0 {
		return ctx, fmt.Errorf("no TSL HTML files found in %s", dirPath)
	}

	// Generate the index.html file
	err = generateIndexHTML(dirPath, entries, title)
	if err != nil {
		return ctx, fmt.Errorf("failed to generate index.html: %w", err)
	}

	return ctx, nil
}

// findTSLHtmlFiles scans a directory for TSL HTML files and extracts metadata from them
func findTSLHtmlFiles(dirPath string) ([]TSLIndexEntry, error) {
	var entries []TSLIndexEntry

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-HTML files
		if d.IsDir() || filepath.Ext(path) != ".html" || filepath.Base(path) == "index.html" {
			return nil
		}

		// Get the relative path for the URL
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}

		// Extract metadata from the HTML file
		entry, err := extractMetadataFromHTML(path, relPath)
		if err != nil {
			// Skip files that don't appear to be TSL HTML files
			return nil
		}

		entries = append(entries, entry)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort entries by territory code
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Territory < entries[j].Territory
	})

	return entries, nil
}

// extractMetadataFromHTML reads a TSL HTML file and extracts metadata for the index
func extractMetadataFromHTML(filePath, relPath string) (TSLIndexEntry, error) {
	entry := TSLIndexEntry{
		Filename: filepath.Base(filePath),
		URL:      relPath,
	}

	// Read the HTML file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return entry, err
	}

	// Parse the HTML document
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(content))
	if err != nil {
		return entry, err
	}

	// Extract title
	entry.Title = doc.Find("title").Text()

	// Extract territory from the specific element if available
	if territoryText := doc.Find(".tsl-meta:contains('Territory')").Text(); territoryText != "" {
		// Try to extract the territory code (usually formatted as "Territory: XX")
		if idx := strings.Index(territoryText, "Territory:"); idx != -1 {
			territory := strings.TrimSpace(territoryText[idx+len("Territory:"):])
			// If we got a territory code (usually 2 characters), use it
			if len(territory) == 2 {
				entry.Territory = territory
			} else {
				// Try to get territory from the title
				parts := strings.Split(entry.Title, " - ")
				if len(parts) > 0 {
					entry.Territory = strings.TrimSpace(parts[0])
				}
			}
		}
	} else {
		// Try to extract from title (common format: "[TERRITORY] - Trust Service Status List")
		parts := strings.Split(entry.Title, " - ")
		if len(parts) > 0 {
			entry.Territory = strings.TrimSpace(parts[0])
		}
	}

	// Extract TSL type
	entry.SchemeType = doc.Find(".tsl-meta code").First().Text()

	// Extract sequence number
	seq := doc.Find(".tsl-meta:contains('TSL Sequence')").Text()
	if idx := strings.Index(seq, "TSL Sequence #:"); idx != -1 {
		parts := strings.Split(seq[idx:], "|")
		if len(parts) > 0 {
			entry.Sequence = strings.TrimSpace(strings.TrimPrefix(parts[0], "TSL Sequence #:"))
		}
	}

	// Extract issue date
	issue := doc.Find(".tsl-meta:contains('Issue Date')").Text()
	if idx := strings.Index(issue, "Issue Date:"); idx != -1 {
		parts := strings.Split(issue[idx:], "|")
		if len(parts) > 0 {
			entry.IssueDate = strings.TrimSpace(strings.TrimPrefix(parts[0], "Issue Date:"))
		}
	}

	// Extract next update date
	next := doc.Find(".tsl-meta:contains('Next Update')").Text()
	if idx := strings.Index(next, "Next Update:"); idx != -1 {
		parts := strings.Split(next[idx:], "|")
		if len(parts) > 0 {
			entry.NextUpdate = strings.TrimSpace(strings.TrimPrefix(parts[0], "Next Update:"))
		}
	}

	// Count trust services
	entry.TrustService = doc.Find(".service-card").Length()

	return entry, nil
}

// generateIndexHTML creates an index.html file with links to all TSL HTML files
func generateIndexHTML(dirPath string, entries []TSLIndexEntry, title string) error {
	// Define the HTML template
	const indexTemplate = `<!DOCTYPE html>
<html lang="en" data-theme="light">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ .Title }}</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@picocss/pico@1/css/pico.min.css">
    <style>
        :root {
            --badge-qualified-bg: #27ae60;
            --badge-nonqualified-bg: #f39c12;
            --badge-info-bg: #3498db;
        }
        
        body {
            padding-bottom: 2rem;
        }

        .container {
            max-width: 1400px;
        }

        /* Header Improvements */
        header {
            margin-bottom: 2rem;
        }

        header h1 {
            margin-bottom: 1rem;
            font-size: 2rem;
        }

        /* Stats Cards */
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1rem;
            margin-bottom: 2rem;
        }

        .stat-card {
            padding: 1rem;
            border-radius: 8px;
            background-color: var(--card-background-color);
            border: 1px solid var(--card-border-color);
            text-align: center;
        }

        .stat-card .number {
            font-size: 2rem;
            font-weight: bold;
            color: var(--primary);
            margin-bottom: 0.25rem;
        }

        .stat-card .label {
            font-size: 0.9rem;
            color: var(--muted-color);
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }

        /* Search and Filter */
        .controls {
            display: flex;
            flex-wrap: wrap;
            gap: 1rem;
            margin-bottom: 1.5rem;
            align-items: center;
        }

        .controls input[type="search"] {
            flex: 1;
            min-width: 250px;
            margin-bottom: 0;
        }

        .controls select {
            min-width: 150px;
            margin-bottom: 0;
        }

        .theme-toggle {
            padding: 0.5rem 1rem;
            background: var(--primary);
            color: white;
            border: none;
            border-radius: 5px;
            cursor: pointer;
            white-space: nowrap;
        }

        .theme-toggle:hover {
            opacity: 0.9;
        }

        /* Badge Styles */
        .badge {
            display: inline-block;
            padding: 0.25rem 0.6rem;
            border-radius: 4px;
            font-size: 0.75rem;
            font-weight: 600;
            margin-right: 0.5rem;
            white-space: nowrap;
        }
        
        .badge-country {
            background-color: var(--primary);
            color: white;
        }

        /* Responsive Table */
        .table-wrapper {
            overflow-x: auto;
            margin-bottom: 2rem;
            border-radius: 8px;
            border: 1px solid var(--card-border-color);
        }

        table {
            margin-bottom: 0;
            width: 100%;
        }

        table th {
            position: sticky;
            top: 0;
            background-color: var(--card-background-color);
            z-index: 10;
            white-space: nowrap;
            cursor: pointer;
            user-select: none;
        }

        table th:hover {
            background-color: var(--primary-hover);
        }

        table th::after {
            content: ' ⇅';
            opacity: 0.3;
            font-size: 0.8em;
        }

        table th.sort-asc::after {
            content: ' ↑';
            opacity: 1;
        }

        table th.sort-desc::after {
            content: ' ↓';
            opacity: 1;
        }

        table td {
            vertical-align: middle;
        }

        table tbody tr {
            transition: background-color 0.2s;
        }

        table tbody tr:hover {
            background-color: var(--primary-hover);
        }

        /* Mobile Responsiveness */
        @media (max-width: 768px) {
            .container {
                padding: 1rem;
            }

            header h1 {
                font-size: 1.5rem;
            }

            .stats-grid {
                grid-template-columns: repeat(2, 1fr);
            }

            .stat-card .number {
                font-size: 1.5rem;
            }

            .controls {
                flex-direction: column;
            }

            .controls input[type="search"],
            .controls select {
                width: 100%;
            }

            /* Stack table cells on mobile */
            table {
                font-size: 0.85rem;
            }

            table th,
            table td {
                padding: 0.5rem;
            }

            .badge {
                font-size: 0.7rem;
                padding: 0.2rem 0.4rem;
            }
        }

        /* Dark mode compatibility */
        @media (prefers-color-scheme: dark) {
            :root:not([data-theme="light"]) {
                --badge-qualified-bg: #27ae60;
                --badge-nonqualified-bg: #f39c12;
                --badge-info-bg: #3498db;
            }
        }

        /* Loading/Empty State */
        .empty-state {
            text-align: center;
            padding: 3rem;
            color: var(--muted-color);
        }

        /* Footer */
        footer {
            margin-top: 3rem;
            padding-top: 2rem;
            border-top: 1px solid var(--card-border-color);
            text-align: center;
            color: var(--muted-color);
            font-size: 0.9rem;
        }
    </style>
</head>
<body>
    <main class="container">
        <header>
            <h1>{{ .Title }}</h1>
        </header>

        <!-- Statistics Cards -->
        <div class="stats-grid">
            <div class="stat-card">
                <div class="number">{{ len .Entries }}</div>
                <div class="label">Total TSLs</div>
            </div>
            <div class="stat-card">
                <div class="number" id="total-services">0</div>
                <div class="label">Trust Services</div>
            </div>
            <div class="stat-card">
                <div class="number" id="total-territories">{{ len .Entries }}</div>
                <div class="label">Territories</div>
            </div>
            <div class="stat-card">
                <div class="number">{{ .GeneratedDate }}</div>
                <div class="label">Last Updated</div>
            </div>
        </div>

        <!-- Search and Filter Controls -->
        <div class="controls">
            <input type="search" id="search" placeholder="Search by territory, title, or type..." 
                   aria-label="Search TSLs">
            <select id="filter-type" aria-label="Filter by type">
                <option value="">All Types</option>
            </select>
            <button class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle dark mode">
                🌓 Toggle Theme
            </button>
        </div>

        <!-- TSL Table -->
        <div class="table-wrapper">
            <table id="tsl-table">
                <thead>
                    <tr>
                        <th onclick="sortTable(0)">Territory</th>
                        <th onclick="sortTable(1)">Sequence</th>
                        <th onclick="sortTable(2)">Issue Date</th>
                        <th onclick="sortTable(3)">Next Update</th>
                        <th onclick="sortTable(4)">Services</th>
                        <th onclick="sortTable(5)">Type</th>
                    </tr>
                </thead>
                <tbody id="tsl-tbody">
                    {{ range .Entries }}
                    <tr data-territory="{{ .Territory }}" data-type="{{ .SchemeType }}">
                        <td>
                            <a href="{{ .URL }}">
                                <span class="badge badge-country">{{ .Territory }}</span>
                                <span class="tsl-title">{{ .Title }}</span>
                            </a>
                        </td>
                        <td>{{ .Sequence }}</td>
                        <td>{{ .IssueDate }}</td>
                        <td>{{ .NextUpdate }}</td>
                        <td>{{ .TrustService }}</td>
                        <td><code>{{ .SchemeType }}</code></td>
                    </tr>
                    {{ end }}
                </tbody>
            </table>
        </div>

        <div id="no-results" class="empty-state" style="display: none;">
            <p>No TSLs found matching your search criteria.</p>
        </div>

        <footer>
            <p>
                <strong>Generated by Go-Trust TSL Pipeline</strong><br>
                {{ .GeneratedDate }} • {{ len .Entries }} Trust Status Lists
            </p>
        </footer>
    </main>

    <script>
        // Calculate total services
        document.addEventListener('DOMContentLoaded', function() {
            let totalServices = 0;
            document.querySelectorAll('#tsl-tbody tr').forEach(row => {
                const services = parseInt(row.cells[4].textContent) || 0;
                totalServices += services;
            });
            document.getElementById('total-services').textContent = totalServices.toLocaleString();

            // Populate type filter
            const types = new Set();
            document.querySelectorAll('#tsl-tbody tr').forEach(row => {
                const type = row.getAttribute('data-type');
                if (type) types.add(type);
            });
            const filterSelect = document.getElementById('filter-type');
            Array.from(types).sort().forEach(type => {
                const option = document.createElement('option');
                option.value = type;
                option.textContent = type;
                filterSelect.appendChild(option);
            });
        });

        // Search functionality
        document.getElementById('search').addEventListener('input', filterTable);
        document.getElementById('filter-type').addEventListener('change', filterTable);

        function filterTable() {
            const searchTerm = document.getElementById('search').value.toLowerCase();
            const filterType = document.getElementById('filter-type').value;
            const rows = document.querySelectorAll('#tsl-tbody tr');
            let visibleCount = 0;

            rows.forEach(row => {
                const text = row.textContent.toLowerCase();
                const type = row.getAttribute('data-type');
                const matchesSearch = text.includes(searchTerm);
                const matchesType = !filterType || type === filterType;

                if (matchesSearch && matchesType) {
                    row.style.display = '';
                    visibleCount++;
                } else {
                    row.style.display = 'none';
                }
            });

            document.getElementById('no-results').style.display = visibleCount === 0 ? 'block' : 'none';
        }

        // Table sorting
        let sortColumn = -1;
        let sortAscending = true;

        function sortTable(columnIndex) {
            const table = document.getElementById('tsl-table');
            const tbody = document.getElementById('tsl-tbody');
            const rows = Array.from(tbody.querySelectorAll('tr'));

            // Update sort direction
            if (sortColumn === columnIndex) {
                sortAscending = !sortAscending;
            } else {
                sortAscending = true;
                sortColumn = columnIndex;
            }

            // Remove sort classes from all headers
            table.querySelectorAll('th').forEach(th => {
                th.classList.remove('sort-asc', 'sort-desc');
            });

            // Add sort class to current header
            const header = table.querySelectorAll('th')[columnIndex];
            header.classList.add(sortAscending ? 'sort-asc' : 'sort-desc');

            // Sort rows
            rows.sort((a, b) => {
                let aValue = a.cells[columnIndex].textContent.trim();
                let bValue = b.cells[columnIndex].textContent.trim();

                // Extract numeric values from badges
                if (columnIndex === 0) {
                    aValue = a.getAttribute('data-territory') || aValue;
                    bValue = b.getAttribute('data-territory') || bValue;
                }

                // Try to parse as numbers
                const aNum = parseFloat(aValue.replace(/[^0-9.-]/g, ''));
                const bNum = parseFloat(bValue.replace(/[^0-9.-]/g, ''));

                if (!isNaN(aNum) && !isNaN(bNum)) {
                    return sortAscending ? aNum - bNum : bNum - aNum;
                }

                // String comparison
                return sortAscending ? 
                    aValue.localeCompare(bValue) : 
                    bValue.localeCompare(aValue);
            });

            // Reorder rows in DOM
            rows.forEach(row => tbody.appendChild(row));
        }

        // Theme toggle
        function toggleTheme() {
            const html = document.documentElement;
            const currentTheme = html.getAttribute('data-theme');
            const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
            html.setAttribute('data-theme', newTheme);
            localStorage.setItem('theme', newTheme);
        }

        // Load saved theme
        document.addEventListener('DOMContentLoaded', function() {
            const savedTheme = localStorage.getItem('theme') || 
                (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
            document.documentElement.setAttribute('data-theme', savedTheme);
        });
    </script>
</body>
</html>`

	// Prepare template data
	data := struct {
		Title         string
		Entries       []TSLIndexEntry
		GeneratedDate string
	}{
		Title:         title,
		Entries:       entries,
		GeneratedDate: time.Now().Format("2006-01-02 15:04:05"),
	}

	// Parse and execute the template
	tmpl, err := template.New("index").Parse(indexTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create the index.html file
	file, err := os.Create(filepath.Join(dirPath, "index.html"))
	if err != nil {
		return fmt.Errorf("failed to create index.html: %w", err)
	}
	defer file.Close()

	// Execute the template and write to the file
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

func init() {
	// Register the GenerateIndex function
	RegisterFunction("generate_index", GenerateIndex)
}
