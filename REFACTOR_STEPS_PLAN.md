# Refactoring Plan: Split steps.go into Smaller Files

## Current Situation
- `steps.go` is 1539 lines - too large for easy maintenance
- Contains multiple concerns: registry, generation, loading, selection, publishing, logging

## Proposed File Structure

### 1. `step_registry.go` (Lines 1-73)
**Purpose:** Step function registration and lookup mechanism

**Contents:**
- `StepFunc` type definition
- `functionRegistry` variable
- `registryMutex` variable
- `RegisterFunction()` function
- `GetFunctionByName()` function

**Justification:** Core registry mechanism used by all steps

---

### 2. `step_generate.go` (Lines 74-540)  
**Purpose:** TSL generation from directory-based metadata

**Contents:**
- `MultiLangName`, `Address`, `ProviderMetadata`, `CertificateMetadata`, `SchemeMetadata` types
- `loadSchemeMetadata()` function
- `loadProviderMetadata()` function  
- `addProviderCertificates()` function
- `GenerateTSL()` step function

**Justification:** Self-contained TSL generation logic with its own types and helpers

---

### 3. `step_load.go` (Lines 541-733)
**Purpose:** Loading TSLs from files and URLs

**Contents:**
- `LoadTSL()` step function
- TSL fetching logic with HTTP/HTTPS support
- Reference following with depth control

**Justification:** Distinct responsibility for TSL loading

---

### 4. `step_fetch_options.go` (Lines 734-852)
**Purpose:** Configuration of TSL fetch behavior

**Contents:**
- `SetFetchOptions()` step function
- Parsing of fetch option arguments (max-depth, timeout, filters, etc.)

**Justification:** Focused on fetch configuration

---

### 5. `step_select.go` (Lines 853-1148)
**Purpose:** Certificate pool selection and filtering

**Contents:**
- `SelectCertPool()` step function
- Certificate extraction logic
- Filtering by service type, status, territory
- Policy-based selection

**Justification:** Complex certificate selection logic deserves its own file

---

### 6. `step_log.go` (Lines 1149-1237)
**Purpose:** Logging step for pipeline debugging

**Contents:**
- `Log()` step function
- Format string parsing
- Context value interpolation

**Justification:** Simple, standalone logging functionality

---

### 7. `step_publish.go` (Lines 1238-1527)
**Purpose:** Publishing TSLs to files with optional signing

**Contents:**
- `PublishTSL()` step function
- XML marshalling
- File-based signing
- PKCS#11 HSM signing
- Tree structure publishing

**Justification:** Complex publishing logic with signing support

---

### 8. `step_echo.go` (Extract from current)
**Purpose:** Simple echo/debug step

**Contents:**
- `Echo()` step function (if it exists)

**Justification:** Utility function for testing

---

### 9. `steps_init.go` (Lines 1528-1539)
**Purpose:** Registration of all pipeline steps

**Contents:**
- `init()` function that registers all steps

**Justification:** Central registration point, imports all step files

---

## Implementation Strategy

### Phase 1: Preparation
1. ✅ Ensure all tests pass
2. ✅ Create this refactoring plan
3. Create backup branch: `git checkout -b refactor-split-steps`

### Phase 2: Extract Steps (One at a time)
For each step file:
1. Create new file with appropriate content
2. Remove duplicated content from `steps.go`
3. Run tests after each extraction
4. Commit if tests pass

### Phase 3: Final Cleanup
1. Remove empty sections from `steps.go`
2. Update imports across all files
3. Run full test suite
4. Update documentation if needed

### Phase 4: Verification
1. Run `go test ./...` - all tests must pass
2. Run `go build` - must compile without errors
3. Check test coverage remains the same
4. Review git diff for sanity

## Benefits

1. **Maintainability:** Easier to find and modify specific step logic
2. **Readability:** Each file has a single, clear purpose
3. **Testing:** Can focus tests on specific functionality
4. **Collaboration:** Multiple developers can work on different steps
5. **Code Review:** Smaller, focused changes are easier to review

## Risks & Mitigation

**Risk:** Breaking existing tests
- **Mitigation:** Extract one file at a time, test after each

**Risk:** Import cycle issues
- **Mitigation:** Keep types in appropriate files, use interfaces if needed

**Risk:** Lost git history
- **Mitigation:** Use `git mv` where possible, maintain commit granularity

## Next Steps

1. Create feature branch
2. Start with simplest extraction (step_log.go)
3. Progress to more complex files
4. Monitor test coverage throughout

## Notes

- All new files should maintain the same package: `package pipeline`
- Keep function names and signatures unchanged to avoid breaking changes
- Update any documentation that references file names
- Consider adding a steps_test.go if integration tests are needed
