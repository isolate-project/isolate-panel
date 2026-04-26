# Fix Plan V2 — Isolate Panel

## P0 — Critical — DONE

### 1. Password auto-generation — DONE
- [x] Backend: Remove `validate:"required"` from `CreateUserRequest.Password`
- [x] Backend: Auto-generate 16-char hex password if empty (like UUID/tokens)
- [x] Backend: Remove redundant `len(password) < 6` check
- [x] Frontend: No changes needed (field intentionally absent from form)

### 2. expiry_date → expiry_days — DONE
- [x] Frontend: Replace text field `expiry_date` with number field `expiry_days`
- [x] Frontend: Add "Unlimited subscription" checkbox — hides/nullifies `expiry_days`
- [x] Frontend: Update Zod schema in `validators.ts`
- [x] Frontend: Update `UserForm.tsx` initial values and form fields

### 3. Tailwind v4 migration — DONE
- [x] Convert `tokens.css` from space-separated RGB to hex colors
- [x] Migrate `tailwind.config.js` → `@theme` block in `index.css` (v4 native)
- [x] Delete `tailwind.config.js`
- [x] Add `@utility` blocks for z-index (z-modal, z-dropdown, etc.) and transition-base
- [x] Add `@variant dark` for `[data-theme="dark"]` selector
- [x] Replace `animate-in fade-in zoom-in-95` with `animate-modalIn` CSS animation
- [x] Add `animate-slideInRight` to Drawer

### 4. Transparent modals — DONE
- [x] `bg-bg-primary` resolves correctly via `@theme` → `var(--bg-primary)` → `#ffffff`/`#09090b`
- [x] `z-modal` generates `z-index: 1050` via `@utility`
- [x] Modal and Drawer both verified

---

## P1 — Important — DONE

### 5. Slider component — DONE
- [x] Created `Slider.tsx` with styled range input, fill track, format labels
- [x] Added CSS for slider thumb (webkit + moz)
- [x] Added `range` type to `FormField` with min/max/step/formatLabel props
- [x] Applied to Settings: JWT access TTL, refresh TTL, max login attempts

### 6. inbound_ids in user form — DONE
- [x] Added multi-select checkbox list for inbounds in UserForm
- [x] Fetches inbounds via `useInbounds()` hook
- [x] Shows inbound name, protocol, port
- [x] Sends `inbound_ids` array in create/update payload
- [x] `EditUserFormWrapper` in Users.tsx fetches current user inbounds for edit

### 7. Core status polling — DONE
- [x] `useCores()` refetchInterval: 5000ms
- [x] `useCoreStatus()` refetchInterval: 5000ms

### 8. traffic_limit_bytes UX — DONE
- [x] GB/MB toggle buttons in user form
- [x] Display value shows conversion to GB
- [x] Converts to bytes on submit

---

## P2 — Improvements (TODO)

### 9. Core health-check
- After StartCore verify process is listening on expected port

### 10. Backend password validation cleanup
- Already done as part of P0-1 (removed len < 6 check, replaced with auto-generation)

### 11. User creation E2E test
- Full flow: create → verify in list → check credentials
