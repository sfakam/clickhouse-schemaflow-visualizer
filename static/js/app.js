const databaseTree = document.getElementById('database-tree');
const refreshBtn = document.getElementById('refresh-btn');
const currentSelection = document.getElementById('current-selection');
const schemaDiagram = document.getElementById('schema-diagram');
const exportHtmlBtn = document.getElementById('export-html-btn');
const zoomInBtn = document.getElementById('zoom-in-btn');
const zoomOutBtn = document.getElementById('zoom-out-btn');
const resetZoomBtn = document.getElementById('reset-zoom-btn');
const tableDetailsContainer = document.querySelector('.table-details-container');
const tableDetailsContent = document.getElementById('table-details');
const toggleTableDetailsBtn = document.getElementById('toggle-table-details');

let databases = [];
let selectedDatabase = null;
let selectedTable = null;
let currentSchema = null;
let currentZoomLevel = 1; // Default zoom level

document.addEventListener('DOMContentLoaded', () => {
    // Initialize Mermaid once at startup
    mermaid.initialize({
        startOnLoad: false,
        theme: 'base',
        themeVariables: {
            primaryColor: '#ECECFF',
            primaryTextColor: '#333333',
            primaryBorderColor: '#9370DB',
            lineColor: '#333333',
            secondaryColor: '#ffffff',
            tertiaryColor: '#ffffff',
            background: '#ffffff',
            fontFamily: 'trebuchet ms, verdana, arial, sans-serif',
            fontSize: '12px'
        },
        securityLevel: 'loose',
        flowchart: {
            useMaxWidth: true,
            htmlLabels: true,
            padding: 15,
            nodeSpacing: 50,
            rankSpacing: 80,
            curve: 'linear'
        },
        er: {
            diagramPadding: 20,
            layoutDirection: 'TB',
            minEntityWidth: 150,
            minEntityHeight: 100,
            entityPadding: 20
        }
    });

    loadDatabases();

    refreshBtn.addEventListener('click', loadDatabases);
    exportHtmlBtn.addEventListener('click', exportHtml);
    
    zoomInBtn.addEventListener('click', zoomIn);
    zoomOutBtn.addEventListener('click', zoomOut);
    resetZoomBtn.addEventListener('click', resetZoom);
    
    // Setup view mode selector
    const viewModeRadios = document.querySelectorAll('input[name="viewMode"]');
    viewModeRadios.forEach(radio => {
        radio.addEventListener('change', handleViewModeChange);
    });
    
    // Initialize database click hints
    updateDatabaseClickHints('table'); // Default to table view
    
    // Setup engine filters
    const applyFiltersBtn = document.getElementById('apply-filters-btn');
    if (applyFiltersBtn) {
        applyFiltersBtn.addEventListener('click', applyEngineFilters);
    }
    
    // Setup collapsible Database header section
    const databaseHeader = document.getElementById('database-header');
    const sidebar = document.querySelector('.sidebar');
    if (databaseHeader && sidebar) {
        databaseHeader.addEventListener('click', () => {
            databaseHeader.classList.toggle('collapsed');
            sidebar.classList.toggle('database-collapsed');
            // Save collapsed state to localStorage
            const isCollapsed = databaseHeader.classList.contains('collapsed');
            localStorage.setItem('databaseHeaderCollapsed', isCollapsed);
        });
        
        // Restore collapsed state from localStorage
        const savedDatabaseState = localStorage.getItem('databaseHeaderCollapsed');
        if (savedDatabaseState === 'true') {
            databaseHeader.classList.add('collapsed');
            sidebar.classList.add('database-collapsed');
        }
    }
    
    // Setup collapsible Table Types section
    const tableTypesHeader = document.querySelector('.legend-container .collapsible-header');
    if (tableTypesHeader) {
        tableTypesHeader.addEventListener('click', () => {
            tableTypesHeader.classList.toggle('collapsed');
            // Save collapsed state to localStorage
            const isCollapsed = tableTypesHeader.classList.contains('collapsed');
            localStorage.setItem('tableTypesCollapsed', isCollapsed);
        });
        
        // Restore collapsed state from localStorage
        const savedCollapsedState = localStorage.getItem('tableTypesCollapsed');
        if (savedCollapsedState === 'true') {
            tableTypesHeader.classList.add('collapsed');
        }
    }
    
    // Setup metadata toggle
    const metadataToggle = document.getElementById('metadata-toggle');
    if (metadataToggle) {
        metadataToggle.addEventListener('change', toggleMetadataVisibility);
        
        // Restore metadata visibility state from localStorage (default is hidden)
        const savedMetadataState = localStorage.getItem('metadataVisible');
        const isVisible = savedMetadataState === 'true';
        metadataToggle.checked = isVisible;
        updateMetadataVisibility(isVisible);
    }
    
    // Setup table details toggle
    const tableDetailsHeader = document.querySelector('.table-details-header');
    if (tableDetailsHeader) {
        tableDetailsHeader.addEventListener('click', toggleTableDetails);
        
        // Restore table details visibility state from localStorage
        const savedTableDetailsState = localStorage.getItem('tableDetailsVisible');
        const isVisible = savedTableDetailsState !== 'false'; // Default to visible
        if (!isVisible) {
            tableDetailsContainer.classList.add('collapsed');
        }
    }
});

async function loadDatabases() {
    try {
        const response = await fetch('/api/databases');
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        databases = await response.json();
        renderDatabaseTree();
    } catch (error) {
        console.error('Error loading databases:', error);
        showError('Failed to load databases. Please check your connection to ClickHouse.');
    }
}

function renderDatabaseTree() {
    databaseTree.innerHTML = '';

    if (typeof databases === 'object' && !Array.isArray(databases)) {
        Object.entries(databases).forEach(([dbName, dbContent]) => {
            const dbItem = document.createElement('li');

            const dbSpan = document.createElement('span');
            dbSpan.className = 'database';
            
            // Get table count for this database
            const tableCount = typeof dbContent === 'object' && !Array.isArray(dbContent) ? Object.keys(dbContent).length : 0;
            dbSpan.textContent = dbName;
            dbSpan.dataset.count = tableCount;
            
            dbSpan.addEventListener('click', (e) => {
                // Check current view mode
                const selectedViewMode = document.querySelector('input[name="viewMode"]:checked').value;
                
                if (selectedViewMode === 'database') {
                    // In database view mode, select database for schema view
                    selectDatabase(dbName);
                } else {
                    // In table view mode, toggle database tables
                    toggleDatabase(dbItem);
                }
            });

            dbItem.appendChild(dbSpan);

            const tablesList = document.createElement('ul');
            tablesList.style.display = 'none';

            if (typeof dbContent === 'object' && !Array.isArray(dbContent) && !dbContent.tables) {
                Object.entries(dbContent).forEach(([dbTable, tableName]) => {
                    addTableToList(tablesList, dbName, dbTable, tableName);
                });
            }

            dbItem.appendChild(tablesList);
            databaseTree.appendChild(dbItem);
        });
    } else if (Array.isArray(databases)) {
        databases.forEach(db => {
            const dbItem = document.createElement('li');
            const dbSpan = document.createElement('span');
            dbSpan.className = 'database';
            
            // Count tables in this database
            let tableCount = 0;
            if (db.tables) {
                if (Array.isArray(db.tables)) {
                    tableCount = db.tables.length;
                } else if (typeof db.tables === 'object') {
                    tableCount = Object.keys(db.tables).length;
                }
            } else if (typeof db === 'object') {
                tableCount = Object.keys(db).filter(key => key !== 'name').length;
            }
            
            dbSpan.textContent = db.name || db.toString();
            dbSpan.dataset.count = tableCount;
            dbSpan.addEventListener('click', (e) => {
                const dbName = db.name || db.toString();
                
                // Check current view mode
                const selectedViewMode = document.querySelector('input[name="viewMode"]:checked').value;
                
                if (selectedViewMode === 'database') {
                    // In database view mode, select database for schema view
                    selectDatabase(dbName);
                } else {
                    // In table view mode, toggle database tables
                    toggleDatabase(dbItem);
                }
            });
            dbItem.appendChild(dbSpan);

            const tablesList = document.createElement('ul');
            tablesList.style.display = 'none';

            if (db.tables) {
                if (Array.isArray(db.tables)) {
                    db.tables.forEach(table => {
                        const tableName = typeof table === 'string' ? table : table.name;
                        addTableToList(tablesList, db.name, tableName);
                    });
                } else if (typeof db.tables === 'object') {
                    Object.keys(db.tables).forEach(tableName => {
                        addTableToList(tablesList, db.name, tableName);
                    });
                }
            } else if (typeof db === 'object') {
                const dbName = db.name || db.toString();
                Object.keys(db)
                    .filter(key => key !== 'name')
                    .forEach(tableName => {
                        addTableToList(tablesList, dbName, tableName);
                    });
            }

            dbItem.appendChild(tablesList);
            databaseTree.appendChild(dbItem);
        });
    } else {
        console.error('Unexpected databases structure:', databases);
        showError('The database structure is not in the expected format.');
    }
    
    // Update database click hints based on current view mode
    const currentViewMode = document.querySelector('input[name="viewMode"]:checked').value;
    updateDatabaseClickHints(currentViewMode);
}

function addTableToList(tablesList, dbName, dbTable, showTableName) {
    const tableItem = document.createElement('li');
    tableItem.className = 'table';
    tableItem.innerHTML = showTableName;
    tableItem.dataset.database = dbName;
    tableItem.dataset.table = dbTable;
    tableItem.title = dbTable;

    tableItem.addEventListener('click', () => selectTable(tableItem));

    tablesList.appendChild(tableItem);
}

function toggleDatabase(dbItem) {
    const tablesList = dbItem.querySelector('ul');
    if (tablesList.style.display === 'none') {
        tablesList.style.display = 'block';
    } else {
        tablesList.style.display = 'none';
    }
}

async function selectDatabase(database) {
    // Clear any selected table
    const previouslySelected = document.querySelector('.table.selected');
    if (previouslySelected) {
        previouslySelected.classList.remove('selected');
    }
    
    // Highlight selected database
    const previouslySelectedDb = document.querySelector('.database.selected');
    if (previouslySelectedDb) {
        previouslySelectedDb.classList.remove('selected');
    }
    
    const allDbSpans = document.querySelectorAll('.database');
    allDbSpans.forEach(span => {
        if (span.textContent.trim() === database) {
            span.classList.add('selected');
        }
    });
    
    selectedDatabase = database;
    
    // Switch to database view if not already
    const databaseRadio = document.getElementById('view-database');
    if (databaseRadio && !databaseRadio.checked) {
        databaseRadio.checked = true;
        handleViewModeChange();
    }
    
    // Load the database schema
    await loadDatabaseSchema(database);
}

async function selectTable(tableItem) {
    const previouslySelected = document.querySelector('.table.selected');
    if (previouslySelected) {
        previouslySelected.classList.remove('selected');
    }

    tableItem.classList.add('selected');

    const database = tableItem.dataset.database;
    const table = tableItem.dataset.table;
    
    selectedDatabase = database;
    selectedTable = table;

    // Reset zoom level when selecting a new table
    currentZoomLevel = 1;

    const selectedViewMode = document.querySelector('input[name="viewMode"]:checked').value;
    
    if (selectedViewMode === 'database') {
        // In database view, selecting a table switches to table view
        document.querySelector('input[name="viewMode"][value="table"]').checked = true;
        handleViewModeChange({target: {value: 'table'}});
    }

    currentSelection.textContent = `${database} / ${table}`;

    await loadTableSchema();
    await loadTableDetails(database, table);
}

async function loadTableSchema() {
    if (!selectedDatabase || !selectedTable) return;

    try {
        const response = await fetch(`/api/schema/${selectedDatabase}/${selectedTable}`);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const data = await response.json();
        currentSchema = data.schema;

        // Render the schema
        renderSchema();
    } catch (error) {
        console.error('Error loading table schema:', error);
        showError('Failed to load table schema.');
    }
}

function formatMermaidSchema(schema) {
    if (!schema || typeof schema !== 'string') return schema;
    return schema;
}

function renderSchema() {
    if (!currentSchema) {
        return;
    }

    const formattedSchema = formatMermaidSchema(currentSchema);

    if (typeof mermaid === 'undefined') {
        showError("Diagram library is loading. Please wait a moment and try again.");

        const mermaidRenderInterval = setInterval(() => {
            if (typeof mermaid !== 'undefined') {
                clearInterval(mermaidRenderInterval);
                try {
                    renderMermaidDiagram(formattedSchema);
                } catch (error) {
                    showError("Failed to render diagram.");
                }
            }
        }, 100);
        return;
    }

    try {
        renderMermaidDiagram(formattedSchema);
    } catch (error) {
        showError("Failed to render diagram.");
    }
}

function renderMermaidDiagram(schema) {
    schemaDiagram.innerHTML = '';

    const container = document.createElement('div');
    container.className = 'mermaid';
    container.textContent = schema;
    schemaDiagram.appendChild(container);

    try {
        // Use the modern mermaid.run() method instead of deprecated mermaid.init()
        mermaid.run({
            querySelector: '.mermaid'
        });
        
        // Check if the SVG was properly rendered
        setTimeout(() => {
            const svg = schemaDiagram.querySelector('svg');
            if (svg) {
                const width = svg.style.maxWidth || svg.getAttribute('width');
                if (width === '16px' || !svg.querySelector('.nodes').hasChildNodes()) {
                    showRawSchema(schema);
                }
            }
        }, 100);
        
        applyZoom();
        
        setupMouseWheelZoom();
    } catch (error) {
        // Fallback to show raw schema
        showRawSchema(schema);
    }
}

function showRawSchema(schema) {
    schemaDiagram.innerHTML = '';
    const rawSchemaDisplay = document.createElement('pre');
    rawSchemaDisplay.style.whiteSpace = 'pre-wrap';
    rawSchemaDisplay.style.fontFamily = 'monospace';
    rawSchemaDisplay.style.padding = '10px';
    rawSchemaDisplay.style.border = '1px solid #ccc';
    rawSchemaDisplay.style.backgroundColor = '#f5f5f5';
    rawSchemaDisplay.textContent = schema;
    schemaDiagram.appendChild(rawSchemaDisplay);
    showError("Failed to render diagram. Showing raw schema instead.");
}

function exportHtml() {
    if (!currentSchema) {
        showError('No schema to export.');
        return;
    }

    const exportSchema = formatMermaidSchema(currentSchema);

    const html = `
        <!DOCTYPE html>
        <html lang="en">
        <head>
            <meta charset="UTF-8">
            <meta name="viewport" content="width=device-width, initial-scale=1.0">
            <title>${selectedDatabase} - ${selectedTable} Schema</title>
            <script src="https://cdn.jsdelivr.net/npm/mermaid@11.6.0/dist/mermaid.min.js" crossorigin="anonymous" defer></script>
            <style>
                body { 
                    font-family: Arial, sans-serif; 
                    margin: 20px;
                    overflow: hidden;
                }
                h1 { color: #2c3e50; }
                .mermaid { 
                    font-family: 'Courier New', Courier, monospace;
                }
                .raw-schema { 
                    white-space: pre-wrap; 
                    font-family: monospace; 
                    padding: 10px; 
                    border: 1px solid #ccc; 
                    margin-top: 20px;
                    display: none;
                }
                .schema-container {
                    position: relative;
                    height: calc(100vh - 100px);
                    overflow: auto;
                    user-select: none;
                    cursor: grab;
                }
                .schema-container:active {
                    cursor: grabbing;
                }
                #schema-diagram {
                    transform-origin: top left;
                    transition: transform 0.2s ease;
                    min-height: 100%;
                    min-width: 100%;
                }
                .view-controls {
                    position: fixed;
                    top: 80px;
                    right: 30px;
                    z-index: 1000;
                    display: flex;
                    gap: 5px;
                    background: rgba(255, 255, 255, 0.9);
                    padding: 5px;
                    border-radius: 5px;
                    box-shadow: 0 2px 5px rgba(0, 0, 0, 0.1);
                }
                .view-controls button {
                    padding: 0.5rem 1rem;
                    margin-left: 0.5rem;
                    background-color: #007bff;
                    color: #fff;
                    border: none;
                    border-radius: 3px;
                    cursor: pointer;
                }
                .view-controls button:hover {
                    background-color: #f0f0f0;
                }
            </style>
        </head>
        <body>
            <h1>${selectedDatabase} - ${selectedTable} Schema</h1>
            <div class="view-controls">
                <button id="zoom-in-btn" title="Zoom in">+</button>
                <button id="zoom-out-btn" title="Zoom out">-</button>
                <button id="reset-zoom-btn" title="Reset zoom">↺</button>
            </div>
            <div class="schema-container">
                <div id="schema-diagram">
                    <pre class="mermaid">
${exportSchema}
                    </pre>
                </div>
            </div>
            <div id="raw-schema" class="raw-schema">
${exportSchema}
            </div>
            <script>
                document.addEventListener('DOMContentLoaded', function() {
                    const rawSchema = document.getElementById('raw-schema');
                    const schemaDiagram = document.getElementById('schema-diagram');
                    const schemaContainer = document.querySelector('.schema-container');
                    const zoomInBtn = document.getElementById('zoom-in-btn');
                    const zoomOutBtn = document.getElementById('zoom-out-btn');
                    const resetZoomBtn = document.getElementById('reset-zoom-btn');
                    
                    let currentZoomLevel = 1;
                    
                    function showRawSchema() {
                        rawSchema.style.display = 'block';
                    }

                    // Zoom functions
                    function zoomIn() {
                        currentZoomLevel = Math.min(currentZoomLevel + 0.1, 20);
                        applyZoom();
                    }

                    function zoomOut() {
                        currentZoomLevel = Math.max(currentZoomLevel - 0.1, 0.5);
                        applyZoom();
                    }

                    function resetZoom() {
                        currentZoomLevel = 1;
                        applyZoom();
                    }

                    function applyZoom() {
                        if (schemaDiagram) {
                            schemaDiagram.style.transform = \`scale(\${currentZoomLevel})\`;
                        }
                    }

                    // Mouse drag functionality
                    let isDragging = false;
                    let startX, startY, scrollLeft, scrollTop;

                    schemaContainer.addEventListener('mousedown', (e) => {
                        isDragging = true;
                        schemaContainer.style.cursor = 'grabbing';
                        startX = e.pageX - schemaContainer.offsetLeft;
                        startY = e.pageY - schemaContainer.offsetTop;
                        scrollLeft = schemaContainer.scrollLeft;
                        scrollTop = schemaContainer.scrollTop;
                    });

                    schemaContainer.addEventListener('mouseleave', () => {
                        isDragging = false;
                        schemaContainer.style.cursor = 'grab';
                    });

                    schemaContainer.addEventListener('mouseup', () => {
                        isDragging = false;
                        schemaContainer.style.cursor = 'grab';
                    });

                    schemaContainer.addEventListener('mousemove', (e) => {
                        if (!isDragging) return;
                        
                        e.preventDefault();
                        const x = e.pageX - schemaContainer.offsetLeft;
                        const y = e.pageY - schemaContainer.offsetTop;
                        
                        const moveX = (x - startX);
                        const moveY = (y - startY);
                        
                        schemaContainer.scrollLeft = scrollLeft - moveX;
                        schemaContainer.scrollTop = scrollTop - moveY;
                    });

                    // Mouse wheel zoom
                    schemaContainer.addEventListener('wheel', (event) => {
                        event.preventDefault();
                        const delta = event.deltaY || event.detail || event.wheelDelta;
                        if (delta < 0) {
                            zoomIn();
                        } else {
                            zoomOut();
                        }
                    }, { passive: false });

                    // Button event listeners
                    zoomInBtn.addEventListener('click', zoomIn);
                    zoomOutBtn.addEventListener('click', zoomOut);
                    resetZoomBtn.addEventListener('click', resetZoom);

                    // Initialize Mermaid
                    if (typeof mermaid !== 'undefined') {
                        try {
                            console.log("Initializing Mermaid in exported HTML");

                            mermaid.initialize({
                                startOnLoad: false,
                                theme: 'default',
                                securityLevel: 'loose',
                                flowchart: {
                                    useMaxWidth: true,
                                    htmlLabels: true
                                },
                                er: {
                                    diagramPadding: 20,
                                    layoutDirection: 'TB',
                                    minEntityWidth: 100,
                                    minEntityHeight: 75,
                                    entityPadding: 15
                                }
                            });

                            try {
                                mermaid.run({ querySelector: '.mermaid' });
                                console.log("Mermaid initialization successful");
                            } catch (renderError) {
                                console.error("Mermaid render error:", renderError);
                                showRawSchema();
                            }
                        } catch (error) {
                            console.error("Error during Mermaid initialization:", error);
                            showRawSchema();
                        }
                    } else {
                        console.error("Mermaid is not defined in exported HTML");
                        showRawSchema();

                        const mermaidCheckInterval = setInterval(function() {
                            if (typeof mermaid !== 'undefined') {
                                clearInterval(mermaidCheckInterval);
                                try {
                                    console.log("Mermaid now available in exported HTML");

                                    mermaid.initialize({
                                        startOnLoad: false,
                                        theme: 'default',
                                        securityLevel: 'loose',
                                        flowchart: {
                                            useMaxWidth: true,
                                            htmlLabels: true
                                        },
                                        er: {
                                            diagramPadding: 20,
                                            layoutDirection: 'TB',
                                            minEntityWidth: 100,
                                            minEntityHeight: 75,
                                            entityPadding: 15
                                        }
                                    });

                                    try {
                                        mermaid.run({ querySelector: '.mermaid' });
                                        console.log("Mermaid initialization successful after waiting");
                                        rawSchema.style.display = 'none';
                                    } catch (renderError) {
                                        console.error("Mermaid render error after waiting:", renderError);
                                        showRawSchema();
                                    }
                                } catch (error) {
                                    console.error("Error during Mermaid initialization after waiting:", error);
                                    showRawSchema();
                                }
                            }
                        }, 100);
                    }
                });
            </script>
        </body>
        </html>
    `;

    // Create a blob and download link
    const blob = new Blob([html], { type: 'text/html' });
    const url = URL.createObjectURL(blob);

    const a = document.createElement('a');
    a.href = url;
    a.download = `${selectedDatabase}_${selectedTable}_schema.html`;
    document.body.appendChild(a);
    a.click();

    // Clean up
    setTimeout(() => {
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    }, 100);
}

function zoomIn() {
    currentZoomLevel = Math.min(currentZoomLevel + 0.1, 20);  // Increased max zoom from 3 to 10
    applyZoom();
}

function zoomOut() {
    currentZoomLevel = Math.max(currentZoomLevel - 0.1, 0.5);
    applyZoom();
}

function resetZoom() {
    currentZoomLevel = 1;
    applyZoom();
}

function applyZoom() {
    if (schemaDiagram) {
        schemaDiagram.style.transform = `scale(${currentZoomLevel})`;
        
        const schemaContainer = document.querySelector('.schema-container');
        if (schemaContainer) {
            if (currentZoomLevel > 1) {
                schemaContainer.style.overflow = 'auto';
            } else {
                schemaContainer.style.overflow = 'auto';
            }
        }
    }
}

function setupMouseWheelZoom() {
    const schemaContainer = document.querySelector('.schema-container');
    if (!schemaContainer) return;
    
    schemaContainer.removeEventListener('wheel', handleMouseWheel);
    schemaContainer.addEventListener('wheel', handleMouseWheel, { passive: false });
    
    // Setup mouse drag scrolling
    let isDragging = false;
    let startX, startY, scrollLeft, scrollTop;
    
    schemaContainer.style.cursor = 'grab';
    
    schemaContainer.addEventListener('mousedown', (e) => {
        isDragging = true;
        schemaContainer.style.cursor = 'grabbing';
        startX = e.pageX - schemaContainer.offsetLeft;
        startY = e.pageY - schemaContainer.offsetTop;
        scrollLeft = schemaContainer.scrollLeft;
        scrollTop = schemaContainer.scrollTop;
    });
    
    schemaContainer.addEventListener('mouseleave', () => {
        isDragging = false;
        schemaContainer.style.cursor = 'grab';
    });
    
    schemaContainer.addEventListener('mouseup', () => {
        isDragging = false;
        schemaContainer.style.cursor = 'grab';
    });
    
    schemaContainer.addEventListener('mousemove', (e) => {
        if (!isDragging) return;
        
        e.preventDefault();
        const x = e.pageX - schemaContainer.offsetLeft;
        const y = e.pageY - schemaContainer.offsetTop;
        
        const moveX = (x - startX);
        const moveY = (y - startY);
        
        schemaContainer.scrollLeft = scrollLeft - moveX;
        schemaContainer.scrollTop = scrollTop - moveY;
    });
    
    console.log("Mouse wheel zoom and drag support set up");
}

function handleMouseWheel(event) {
    event.preventDefault();
    
    const delta = event.deltaY || event.detail || event.wheelDelta;
    
    if (delta < 0) {
        zoomIn();
    } else {
        zoomOut();
    }
    
    console.log(`Zoom level: ${currentZoomLevel.toFixed(1)}`);
}

function toggleMetadataVisibility() {
    const metadataToggle = document.getElementById('metadata-toggle');
    const isVisible = metadataToggle.checked;
    updateMetadataVisibility(isVisible);
    localStorage.setItem('metadataVisible', isVisible);
}

function updateMetadataVisibility(isVisible) {
    const sidebar = document.querySelector('.sidebar');
    if (sidebar) {
        if (isVisible) {
            sidebar.classList.add('metadata-visible');
        } else {
            sidebar.classList.remove('metadata-visible');
        }
    }
}

function toggleTableDetails() {
    tableDetailsContainer.classList.toggle('collapsed');
    const isVisible = !tableDetailsContainer.classList.contains('collapsed');
    localStorage.setItem('tableDetailsVisible', isVisible);
    
    // Update button icon rotation
    const icon = toggleTableDetailsBtn.querySelector('i');
    if (icon) {
        icon.style.transform = isVisible ? 'rotate(90deg)' : 'rotate(0deg)';
    }
}

async function loadTableDetails(database, table) {
    if (!database || !table) return;

    try {
        const response = await fetch(`/api/table/${database}/${table}`);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const tableDetails = await response.json();
        renderTableDetails(tableDetails);
    } catch (error) {
        console.error('Error loading table details:', error);
        showTableDetailsError('Failed to load table details.');
    }
}

function renderTableDetails(details) {
    if (!details) {
        showTableDetailsError('No table details available.');
        return;
    }

    const formatBytes = (bytes) => {
        if (!bytes) return 'N/A';
        const units = ['B', 'KB', 'MB', 'GB', 'TB'];
        let size = bytes;
        let unitIndex = 0;
        while (size >= 1024 && unitIndex < units.length - 1) {
            size /= 1024;
            unitIndex++;
        }
        return `${size.toFixed(1)} ${units[unitIndex]}`;
    };

    const formatRows = (rows) => {
        if (!rows) return 'N/A';
        return rows.toLocaleString();
    };

    const html = `
        <div class="table-info">
            <h4><i class="fa-solid fa-info-circle"></i> Table Information</h4>
            <div class="table-info-grid">
                <span class="table-info-label">Name:</span>
                <span class="table-info-value">${details.name}</span>
                <span class="table-info-label">Database:</span>
                <span class="table-info-value">${details.database}</span>
                <span class="table-info-label">Engine:</span>
                <span class="table-info-value">${details.engine}</span>
                <span class="table-info-label">Rows:</span>
                <span class="table-info-value">${formatRows(details.total_rows)}</span>
                <span class="table-info-label">Size:</span>
                <span class="table-info-value">${formatBytes(details.total_bytes)}</span>
            </div>
        </div>
        
        <div class="columns-section">
            <h4><i class="fa-solid fa-columns"></i> Columns (${details.columns.length})</h4>
            <table class="columns-table">
                <thead>
                    <tr>
                        <th>Name</th>
                        <th>Type</th>
                    </tr>
                </thead>
                <tbody>
                    ${details.columns.map(column => `
                        <tr>
                            <td class="column-name">${column.name}</td>
                            <td><span class="column-type">${column.type}</span></td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        </div>
    `;

    tableDetailsContent.innerHTML = html;
}

function showTableDetailsError(message) {
    tableDetailsContent.innerHTML = `
        <div class="no-table-selected">
            <i class="fa-solid fa-exclamation-triangle"></i>
            <p>${message}</p>
        </div>
    `;
}

function showNoTableSelected() {
    tableDetailsContent.innerHTML = '<p class="no-table-selected">Select a table to view its details</p>';
}

function showError(message) {
    alert(message);
}

// Handle view mode changes (Table vs Database view)
function handleViewModeChange(event) {
    const viewMode = event ? event.target.value : document.querySelector('input[name="viewMode"]:checked').value;
    const engineFilters = document.getElementById('engine-filters');
    
    // Update database click behavior hints
    updateDatabaseClickHints(viewMode);
    
    if (viewMode === 'database') {
        engineFilters.style.display = 'flex';
        // Switch to database view mode
        if (selectedDatabase) {
            loadDatabaseSchema(selectedDatabase);
        }
    } else {
        engineFilters.style.display = 'none';
        // Switch back to table view mode
        if (selectedDatabase && selectedTable) {
            loadTableSchema(selectedDatabase, selectedTable);
        }
    }
}

// Update database click hints based on view mode
function updateDatabaseClickHints(viewMode) {
    const databases = document.querySelectorAll('.database');
    databases.forEach(db => {
        if (viewMode === 'database') {
            db.title = 'Click to view database schema';
            db.style.cursor = 'pointer';
        } else {
            db.title = 'Click to expand/collapse tables';
            db.style.cursor = 'pointer';
        }
    });
}

// Apply engine filters for database view
function applyEngineFilters() {
    if (!selectedDatabase) {
        showError('Please select a database first');
        return;
    }
    
    const selectedViewMode = document.querySelector('input[name="viewMode"]:checked').value;
    if (selectedViewMode === 'database') {
        loadDatabaseSchema(selectedDatabase);
    }
}

// Load database-level schema with filters
async function loadDatabaseSchema(database) {
    try {
        // Get selected engine filters
        const engineCheckboxes = document.querySelectorAll('.engine-checkboxes input[type="checkbox"]:checked');
        const selectedEngines = Array.from(engineCheckboxes).map(cb => cb.value);
        
        // Get metadata preference
        const metadataToggle = document.getElementById('metadata-toggle');
        const includeMetadata = metadataToggle ? metadataToggle.checked : true;
        
        // Build query parameters
        const params = new URLSearchParams();
        selectedEngines.forEach(engine => params.append('engines', engine));
        params.append('metadata', includeMetadata.toString());
        
        const response = await fetch(`/api/database/${database}/schema?${params.toString()}`);
        
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        const data = await response.json();
        
        // Update current selection display - just show database name
        currentSelection.textContent = `Database: ${database}`;
        
        // Clear table details for database view
        showNoTableSelected();
        
        // Set the current schema and render
        currentSchema = data.schema;
        await renderSchema();
        
    } catch (error) {
        showError(`Failed to load database schema: ${error.message}`);
    }
}
