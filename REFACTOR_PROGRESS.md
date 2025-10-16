# Steps.go Refactoring Progress

## Completed ✅

### Phase 1: Initial Setup
- ✅ Created feature branch `refactor-split-steps`
- ✅ Backed up original file
- ✅ Created refactoring plan document

### Phase 2: Extractions Completed  
- ✅ Extracted `init()` function to `steps_init.go` - Commit: `3aa84fd`
- ✅ Extracted `Echo()` and `Log()` to `step_log.go` - Commit: `8c9a815`
- ✅ Extracted registry to `step_registry.go` - Commit: `092013f`
- ✅ Extracted `SetFetchOptions()` to `step_fetch_options.go` - Commit: `01015c5`
- ✅ Extracted `LoadTSL()` to `step_load.go` - Commit: `f25954e`

**Progress**: 1539 lines → 1074 lines (465 lines extracted into 4 files)
**Status**: All tests passing after each extraction

## Remaining Work 🔄

The remaining extractions follow the same pattern:
1. Create new file with extracted content
2. Add proper package declaration and imports
3. Remove extracted content from steps.go
4. Run tests (`go test ./pkg/pipeline/...`)
5. Commit if tests pass

### Files to Create:

#### 1. `step_registry.go` (Lines 1-73 of original steps.go)
**Extract:**
- `StepFunc` type
- `functionRegistry` variable
- `registryMutex` variable  
- `RegisterFunction()` function
- `GetFunctionByName()` function

**Imports needed:**
```go
import "sync"
```

#### 2. `step_generate.go` (Lines 74-540)
**Extract:**
- `MultiLangName`, `Address`, `ProviderMetadata`, `CertificateMetadata`, `SchemeMetadata` types
- `loadSchemeMetadata()` function
- `loadProviderMetadata()` function
- `addProviderCertificates()` function
- `GenerateTSL()` function

**Imports needed:**
```go
import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SUNET/g119612/pkg/etsi119612"
	"gopkg.in/yaml.v3"
)
```

#### 3. `step_load.go` (Lines 541-733)
**Extract:**
- `LoadTSL()` function

**Imports needed:**
```go
import (
	"fmt"
	"time"

	"github.com/SUNET/g119612/pkg/etsi119612"
)
```

#### 4. `step_fetch_options.go` (Lines 734-852)
**Extract:**
- `SetFetchOptions()` function

**Imports needed:**
```go
import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/SUNET/g119612/pkg/etsi119612"
)
```

#### 5. `step_select.go` (Lines 853-1106)
**Extract:**
- `SelectCertPool()` function

**Imports needed:**
```go
import (
	"crypto/x509"
	"fmt"

	"github.com/SUNET/g119612/pkg/etsi119612"
)
```

#### 6. `step_log.go` (Lines 1107-1237)
**Extract:**
- `Echo()` function
- `Log()` function

**Imports needed:**
```go
import (
	"strings"

	"github.com/SUNET/go-trust/pkg/logging"
)
```

#### 7. `step_publish.go` (Lines 1238-1527)
**Extract:**
- `PublishTSL()` function

**Imports needed:**
```go
import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SUNET/g119612/pkg/etsi119612"
	"github.com/SUNET/go-trust/pkg/dsig"
)
```

## Final Steps

After all extractions:

1. **Verify steps.go is minimal:**
   - Should only contain package declaration and imports
   - Or can be deleted entirely if all content is extracted

2. **Run full test suite:**
   ```bash
   go test ./...
   ```

3. **Check test coverage:**
   ```bash
   go test -cover ./pkg/pipeline/...
   ```

4. **Merge to main:**
   ```bash
   git checkout main
   git merge refactor-split-steps
   ```

5. **Clean up:**
   - Delete `REFACTOR_STEPS_PLAN.md`
   - Delete `refactor_steps.sh`  
   - Delete this progress file

## Testing Checklist

After each extraction, verify:
- [ ] File compiles without errors
- [ ] All pipeline tests pass
- [ ] No duplicate symbol errors
- [ ] Imports are correct and minimal

## Notes

- Each new file should start with `package pipeline`
- Keep function signatures identical to avoid breaking changes
- Private helper functions (lowercase) can stay with their callers
- Public types used across files should be in an appropriate shared location
