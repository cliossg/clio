# Feature "Bridge Area" - Contexto Completo para Continuar

## Directorio Correcto
```bash
cd ~/Projects/clio/clio && claude
```

---

## Feature Solicitada: Bridge Area para Importación de Markdowns

**Concepto**: Un directorio (o colección de directorios) donde puedes dejar archivos `.md` editados en Neovim, y Clio los detecta, lista en un menú específico, y permite importarlos al sistema.

---

## Requisitos Definidos (Ya Respondidos - NO Preguntar de Nuevo)

### 1. Directorio Bridge
- **Default**: `~/Documents/Clio/...`
- **Configurable**: Vía Settings/Attributes
- **Futuro**: Posibilidad de múltiples directorios

### 2. Formato del Markdown
- **Híbrido**: Si tiene frontmatter YAML → usarlo (idealmente compatible con el export de Clio)
- Si NO tiene frontmatter → inferir título del primer `# H1`
- El resto de campos → editables en la UI normal de edición (no wizard)

### 3. Multi-site
- Al importar → preguntar a qué site interno importar
- **Batch**: Seleccionar varios → importarlos todos al mismo site

### 4. Usuario Asociado
- El user logueado = autor de los contenidos importados

### 5. Post-importación (Archivo Original)
- **NO mover** el archivo, dejarlo en su lugar
- Trackear estado de importación en la DB

### 6. Estados en el Listado UI
- Grisado con tag "imported"
- Fecha de importación visible
- Filtrable por JS en cliente

### 7. Lógica de Reimportación (Crítica)
```
SI file_mtime > import_date:
    → Disponible para reimportar

PERO SI content_updated_at > import_date
     AND content_updated_at > file_mtime:
    → NO mostrar como reimportable (proteger ediciones web)
    → Alert: "Este contenido fue modificado en la web después de importarlo"
```

### 8. Reimportación = Actualizar Existente
- NO versionar (no crear nuevo Content)
- Actualiza el Content vinculado
- Alert de conflicto si hubo ediciones web

---

## Información del Proyecto (del Agente Explorador Inicial)

**Stack**: Go + Chi + SQLite + Goldmark + SQLC

**Estructura**:
```
internal/feat/ssg/      → handlers, service, generator, processor
internal/db/sqlc/       → código generado por SQLC
assets/migrations/      → 16 migraciones SQL
assets/queries/         → 11 archivos .sql para SQLC
assets/templates/ssg/   → templates HTML
```

**Modelo Content** (campos relevantes):
- ID, ShortID, SiteID, Heading, Body, Summary, Kind
- Draft, Featured, PublishedAt
- SectionID, UserID, ContributorID
- CreatedAt, UpdatedAt

**Settings**: Sistema flexible por site con RefKey (ej: `ssg.backup.repo.url`)

**Rutas existentes**: `/ssg/list-contents`, `/ssg/new-content`, `/ssg/create-content`, etc.

---

## Lo que Falta Explorar (en el repo correcto)

1. Formato exacto del frontmatter YAML exportado por Clio (buscar en generator.go o backup.go)
2. Ejemplos de migraciones SQL (patrón de naming y estructura)
3. Templates de listado (para replicar UI con checkboxes, filtros)

---

## Diseño Preliminar (a validar con código real)

### Nueva Tabla: `bridge_imports`
```sql
CREATE TABLE bridge_imports (
    id TEXT PRIMARY KEY,
    short_id TEXT NOT NULL,
    file_path TEXT NOT NULL,          -- path absoluto del archivo
    file_hash TEXT,                   -- SHA256 para detectar cambios
    file_mtime DATETIME,              -- última modificación del archivo
    content_id TEXT,                  -- FK a content (NULL si no importado)
    site_id TEXT,                     -- FK a site donde se importó
    user_id TEXT NOT NULL,            -- FK a user que importó
    status TEXT DEFAULT 'pending',    -- pending, imported, conflict
    imported_at DATETIME,             -- cuándo se importó
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (content_id) REFERENCES contents(id),
    FOREIGN KEY (site_id) REFERENCES sites(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

### Nuevos Settings
```
bridge.directories.paths     → JSON array de paths (default: ["~/Documents/Clio"])
bridge.scan.extensions       → extensiones a buscar (default: ".md")
```

### Nuevas Rutas
```
GET  /ssg/bridge/list           → listado de archivos en bridge areas
POST /ssg/bridge/scan           → re-escanear directorios
GET  /ssg/bridge/preview/:id    → preview de un archivo antes de importar
POST /ssg/bridge/import         → importar uno o varios (batch)
POST /ssg/bridge/reimport/:id   → reimportar con detección de conflictos
```

### Lógica de Detección de Estados
```go
func (s *Service) GetBridgeFileStatus(file BridgeFile, import *BridgeImport) string {
    if import == nil {
        return "new"  // nunca importado
    }

    if file.Mtime.Before(import.ImportedAt) {
        return "imported"  // sin cambios desde import
    }

    // El archivo cambió después de importar
    content := s.GetContent(import.ContentID)

    if content.UpdatedAt.After(import.ImportedAt) &&
       content.UpdatedAt.After(file.Mtime) {
        return "conflict"  // editado en web, más reciente que archivo
    }

    return "updated"  // disponible para reimportar
}
```

---

## Instrucciones para Claude en Nueva Sesión

1. **Lee este archivo primero**
2. **NO preguntes los requisitos de nuevo** - ya están definidos arriba
3. **Entra directo a Plan Mode** (`EnterPlanMode`)
4. **Explora**:
   - `internal/feat/ssg/generator.go` o `backup.go` → formato frontmatter
   - `assets/migrations/sqlite/` → patrón de migraciones
   - `assets/templates/ssg/contents/` → patrón de listados
5. **Diseña el plan** basándote en los requisitos y el diseño preliminar de arriba
