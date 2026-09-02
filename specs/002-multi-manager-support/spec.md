# Feature Specification: Multi-Manager Support with Superadmin Bootstrap

**Feature Branch**: `002-multi-manager-support`

**Created**: 2026-09-02

**Status**: Draft

**Input**: User description: "Multi-manager support for Hamsa — role-aware invites (manager vs resident), no silent role promotion on redemption, manager-grant endpoints per building with last-manager protection and audit logging."
**Clarification (2026-09-02)**: "By default, the application should have a superadmin user who can create building manager users. Then, any building manager user can log in and create a building and create another user as a building manager or resident."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Default superadmin bootstraps the deployment (Priority: P1)

As the platform owner installing Hamsa fresh, I want the application to have a
default **superadmin** account — established by the one-time first-run setup —
that can create building-manager users, so the deployment can gain managers
without ever touching the database or server files.

**Why this priority**: Without a bootstrapper, no second privileged account can
ever exist on a fresh install; everything else depends on this.

**Independent Test**: On a fresh deployment, complete first-run setup, confirm
the created account is the superadmin and that setup is now locked; from that
account, create a building-manager user for a phone; confirm the invited phone
can register as a manager and setup cannot run again.

**Acceptance Scenarios**:

1. **Given** a fresh deployment with no accounts, **When** the first account is
   created through setup, **Then** that account is the superadmin and setup
   locks afterwards.
2. **Given** the superadmin is signed in, **When** they create a
   building-manager user for a phone, **Then** a manager-role invitation is
   issued and the invited person's new account is a manager on registration.
3. **Given** the superadmin exists, **When** anyone tries to create another
   superadmin, demote the superadmin, or delete the superadmin through app
   features, **Then** there is no such path — the superadmin is exactly one and
   permanent.
4. **Given** setup has completed, **When** a second setup attempt is made,
   **Then** it is refused (existing lock behavior preserved).

---

### User Story 2 - Managers create buildings and other users (Priority: P1)

As a building manager, I want to log in, create a building, and create other
users as either building managers or residents via invites — so day-to-day
onboarding never needs the superadmin.

**Why this priority**: This is the core multi-manager promise: managers are
self-sufficient once one exists.

**Independent Test**: With a manager account, create a building; issue a
manager-role invite and a resident-role invite; register both phones; confirm
roles. Confirm a resident cannot issue invites and invalid role values are
rejected.

**Acceptance Scenarios**:

1. **Given** I am a signed-in manager, **When** I create a building, **Then**
   I become its manager (existing behavior preserved).
2. **Given** I am a signed-in manager, **When** I request an invite choosing
   the "manager" role, **Then** the invite is created, the response names the
   role, and it expires after the standard 7 days.
3. **Given** I am a signed-in manager, **When** I request an invite without
   choosing a role, **Then** a "resident" invite is created (default).
4. **Given** I am a signed-in manager, **When** I request an invite with any
   role other than "manager" or "resident" (including "superadmin"), **Then**
   the request is rejected with a clear Persian error message.
5. **Given** a manager invite exists for phone P, **When** a brand-new user
   registers with that invite, **Then** the new account has the manager role.
6. **Given** a manager invite exists for phone P, **When** P belongs to an
   existing **resident**, **Then** redemption resets that person's password
   only; their role stays resident. No invite ever silently promotes anyone.
7. **Given** any invite, **When** it is redeemed, **Then** all existing invite
   rules still hold: bound to its phone, single-use, expired after 7 days,
   stored only as a one-way hash.
8. **Given** I am a resident, **When** I try to issue any invite, **Then** the
   request is refused (existing behavior preserved).

---

### User Story 3 - Grant and revoke building-manager access (Priority: P2)

As a current manager of building B, I want to see who manages B, add another
existing user as a manager of B, and remove a manager from B — so building
ownership is shared without touching the database.

**Why this priority**: A newly created manager account manages nothing until
granted; this closes the loop of "create manager → make them useful".

**Independent Test**: With two manager accounts and one building, one manager
lists B's managers, adds the other, is rejected when adding them twice, removes
them, and is rejected when removal would leave B with zero managers or when
they try to remove themselves.

**Acceptance Scenarios**:

1. **Given** I am a manager of building B, **When** I list B's managers,
   **Then** I get each manager's identity, phone, display name, and the date
   their access was granted.
2. **Given** an active user with phone P (resident or manager), **When** I add
   P as a manager of B, **Then** P gains management of B; if P was a resident,
   P's account role becomes manager, and the response reflects the new role.
3. **Given** P already manages B, **When** I add P again, **Then** the request
   is refused as a conflict with the Persian message "این مدیر از قبل دسترسی دارد."
   and nothing changes.
4. **Given** P is not an active user, or P is the superadmin, **When** I add P
   to B, **Then** the request is refused.
5. **Given** B has exactly one manager (me), **When** I try to remove myself,
   **Then** the request is refused with a conflict — I must hand over to
   someone else first. Removing myself via this feature is never allowed.
6. **Given** B has managers {me, P}, **When** I remove P, **Then** P loses
   management of B and B still has at least one manager.
7. **Given** I am a manager of some building but **not** of building X,
   **When** I attempt any of list/add/remove on X, **Then** every attempt is
   refused (forbidden). Managing managers is per-building.
8. **Given** a building has any number of managers and a manager manages any
   number of buildings, **When** any grant/revoke succeeds, **Then** the
   many-to-many relationship is never capped.

---

### User Story 4 - Audit every privilege change (Priority: P3)

As an operator, I want every invite issued with a role, and every manager
grant/revoke, recorded as an audit entry with who did it, the affected party,
and the before/after state — so permission changes are traceable.

**Why this priority**: Governance layer over US1–US3; valuable but the product
functions without it in a demo.

**Independent Test**: Perform an invite issue, a grant, and a revoke; confirm
one audit entry each, with actor, object, action name, and role/user details.

**Acceptance Scenarios**:

1. **Given** a superadmin or manager invites someone with a role, **When** the
   invite is created, **Then** an "invite.issued" audit entry records the
   target phone and role.
2. **Given** a manager is added to a building, **When** the grant succeeds,
   **Then** a "building.manager_granted" audit entry records the actor,
   building, and new manager.
3. **Given** a manager is removed from a building, **When** the revoke
   succeeds, **Then** a "building.manager_revoked" audit entry records the
   actor, building, and removed manager with before/after state.
4. **Given** the audit store is temporarily unavailable, **When** any of the
   above succeed, **Then** the user-facing operation still succeeds (audit is
   best-effort, like login recording today).

---

### User Story 5 - Manager surfaces in the mobile app (Priority: P2)

As a manager (or the superadmin) using the Hamsa app, I want the invite screen
to let me choose ساکن/مدیر (resident/manager) and show which role was issued,
and I want a "managers" screen for each of my buildings to list, add, and
remove managers — in Persian, RTL.

**Why this priority**: The backend capability is unreachable without it; the
superadmin bootstrap and manager self-service both live in the app.

**Independent Test**: Drive the app: toggle role on the invite screen, issue an
invite, see the role in the result banner; open a building's managers screen,
see the list, add by phone, remove with a confirmation dialog.

**Acceptance Scenarios**:

1. **Given** the invite screen, **When** I pick مدیر and submit, **Then** the
   request carries the manager role and the success banner states which role
   was issued.
2. **Given** a building I manage, **When** I open its managers screen (next to
   the units/people entries), **Then** I see current managers with phone and
   grant date, an "افزودن مدیر" phone field, and a remove action with a
   confirm dialog.
3. **Given** a building I do not manage, **When** I view it, **Then** no
   manager-management actions are offered.
4. **Given** any manager API error (conflict, forbidden, validation), **When**
   it occurs in the app, **Then** the Persian error from the server is shown
   through the app's standard error rendering.

### Edge Cases

- Redemption of a manager invite by an existing resident: password reset only,
  no promotion (US2 scenario 6).
- Invite (of any role) to the superadmin's phone: redemption resets the
  superadmin's password only; their role is untouched.
- Adding the superadmin as a building manager: refused — superadmin is not a
  building-management participant.
- Adding an already-granted manager: conflict, idempotent, no duplicate audit
  entry beyond the refusal.
- Adding an inactive (disabled) user's phone: refused.
- Removing the caller's own grant: refused even when other managers exist —
  handover (add successor, then have successor remove you) is the only path.
- Last-manager removal: refused (conflict) so a building never becomes
  unmanaged. If a building ever loses all its people for other reasons, the
  superadmin can create a new manager and the data owner can re-grant; no app
  path bypasses the rule.
- Non-manager (resident) callers of any managers endpoint or invite endpoint:
  refused.
- Manager of building A calling on building B: refused per building.
- Invite role values other than resident/manager (e.g. "superadmin"): refused
  with Persian 400.
- Concurrent duplicate grants: the second attempt is a conflict; storage
  uniqueness is the arbiter.
- Audit write failure: does not roll back the granted/revoked/issued
  operation.

## Requirements *(mandatory)*

### Functional Requirements

#### Roles and the superadmin

- **FR-001**: The system MUST have exactly one superadmin account, established
  by the one-time first-run setup on a fresh deployment; the setup lock
  behavior is otherwise unchanged. *(US1)*
- **FR-002**: There MUST be no path — API or app — to create a second
  superadmin, demote the superadmin, delete the superadmin, grant the
  superadmin a building, or remove the superadmin from any list of manageable
  principals. *(US1, US3)*
- **FR-003**: The superadmin MUST be able to create building-manager users
  (issue manager-role invites); a manager who manages no building is a valid,
  if empty, state until granted one. *(US1)*

#### Invites

- **FR-004**: Invites MUST carry a role (resident or manager); default remains
  resident. *(US2)*
- **FR-005**: Only the superadmin or a manager may issue invites;
  manager-role invites may be issued by either. Any other role value MUST be
  rejected with a Persian 400 message. Residents MUST NOT gain invite ability.
  *(US1, US2)*
- **FR-006**: Registering a new user with an invite MUST assign the invite's
  role. *(US2)*
- **FR-007**: Redeeming an invite for an existing user MUST change only the
  password, never the role — including for the superadmin. Silent promotion or
  demotion via invite is prohibited. *(US1, US2)*
- **FR-008**: All existing invite guarantees MUST remain unchanged:
  phone-bound, single-use, 7-day expiry, stored as a one-way hash. *(US2)*
- **FR-009**: The invite response MUST include the issued role and the expiry
  window so clients can display it. *(US2, US5)*

#### Buildings and grants

- **FR-010**: A manager MUST be able to create a building and automatically
  become its manager (existing behavior preserved). *(US2)*
- **FR-011**: A building's current managers MUST be listable with user id,
  phone, name, and grant date. *(US3)*
- **FR-012**: A current manager of building B MUST be able to add any active
  non-superadmin user (by phone) as a manager of B. *(US3)*
- **FR-013**: Adding a resident as a building manager MUST flip their account
  role to manager, and the response MUST reflect the resulting role.
  Demotion is never performed by this feature. *(US3)*
- **FR-014**: Adding an existing manager of B MUST be refused as a conflict
  with message "این مدیر از قبل دسترسی دارد." *(US3)*
- **FR-015**: A current manager of B MUST be able to remove another manager's
  grant for B. *(US3)*
- **FR-016**: Removal MUST be refused (conflict) when it would leave B with
  zero managers, and the caller MUST never remove their own grant (rejected
  with 400). *(US3)*
- **FR-017**: All three manager-management operations MUST require the caller
  to be a current manager of that specific building; everyone else gets
  forbidden. *(US3)*
- **FR-018**: Manager count per building and buildings per manager remain
  unlimited (many-to-many); nothing in this feature caps either. *(US3)*
- **FR-019**: The manager role is only ever raised explicitly — by
  invite-at-registration or by a building grant from a current manager — and
  never lowered. *(US1–US3)*

#### Audit and errors

- **FR-020**: Issuing an invite with a role, granting a manager, and revoking a
  manager MUST each create an audit entry (actions "invite.issued",
  "building.manager_granted", "building.manager_revoked") recording actor,
  object, and role/user before/after. Audit writes are best-effort and MUST
  NOT fail the operation. *(US4)*
- **FR-021**: All new server errors MUST use the established Persian error
  envelope; all app calls MUST go through the shared API client and standard
  error rendering. *(US1–US5)*

#### Mobile app

- **FR-022**: The mobile invite screen MUST offer a ساکن/مدیر choice, send the
  selected role, and display the issued role in the result banner. *(US5)*
- **FR-023**: The app MUST provide a per-building managers screen (list, add
  by phone, remove with confirm) reachable from building rows, with Persian
  RTL strings, and MUST hide management actions from non-managers. *(US5)*

### Non-Requirements (explicitly out of scope)

- Demoting a manager to resident (no endpoint, no UI).
- Removing yourself as a manager of a building (handover instead).
- Creating, replacing, demoting, or deleting the superadmin after first-run
  setup.
- Superadmin browsing/managing buildings or residents directly; their role is
  bootstrap-of-managers only.
- Self-service building join by non-managers.
- Changing login or general invite lifecycle rules.

### Key Entities

- **Superadmin**: the single, permanent platform-owner account created by
  first-run setup; its only power is creating building-manager users. It does
  not participate in building management.
- **User account**: phone-identified person with a global role
  (resident | building-manager | superadmin); role only ever raised by
  invite-at-creation or by an explicit building grant, never lowered.
- **Invite code**: one-time, phone-bound registration token; now also carries
  the role (resident | manager) the redeemed account will receive.
- **Building manager grant**: the association between a manager and a
  building; many-to-many; created by building creation or by an existing
  manager of that building; removed by an existing manager of that building
  subject to last-manager protection.
- **Audit entry**: who did what to whom/which building, with before/after
  state, for the three sensitive actions above.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A fresh deployment reaches a working two-plus-manager state
  entirely through the app — setup → superadmin creates manager → manager
  creates building → manager grants co-manager — with zero database or
  server-side manual steps.
- **SC-002**: 100% of invite redemptions preserve existing users' roles — no
  redemption ever changes a role; verified by tests covering new-user and
  existing-user redemption of manager invites, including against the
  superadmin's phone.
- **SC-003**: A building can never be observed in a zero-manager state through
  this feature's operations (all removal attempts that would cause it are
  refused).
- **SC-004**: Every successful grant, revoke, and manager/resident invite issue
  has exactly one corresponding audit entry with actor and affected party.
- **SC-005**: A manager without a grant for a building cannot list, add, or
  remove managers for it — 3/3 sub-operations refused, verified in tests; and
  no app-reachable operation ever creates a second superadmin or alters the
  first one's role.
- **SC-006**: Managers can complete add/remove of a co-manager in under a
  minute in the app (list → add → confirm; remove → confirm dialog).

## Assumptions

- The superadmin "exists by default" by repurposing the existing first-run
  setup: the first account created becomes the superadmin instead of a plain
  manager. This avoids seeded default credentials (a security anti-pattern)
  while keeping the setup lock.
- "Create a building manager user" uses the existing one-time invite flow
  (phone-bound code, recipient sets their own password) — no admin-set
  passwords.
- Exactly one superadmin per deployment; recovery of a lost superadmin
  password is out of scope (handled the same way any account reset is today,
  and building-manager invites remain issuable by existing managers, so the
  superadmin is not a daily dependency).
- The superadmin is deliberately excluded from building manager lists
  (cannot be added/removed as a building manager); buildings are governed
  solely by their granted managers.
- Existing "manager" deployments that already have a first account are treated
  as already-bootstrapped: that account behaves as a manager and can issue
  manager invites; no retroactive superadmin is invented for it.
- The existing 7-day invite expiry, phone binding, single-use redemption, and
  hashed-storage rules are the security baseline and are preserved as-is; the
  only invite change is the added role.
- "Active user" means the account exists and is not disabled — the same notion
  the current auth/building checks use.
- Managers discovering future co-managers happens by phone number (same
  assumption as today's resident invite flow); the managers list only exposes
  existing grantees.
- The invite response shape may change outright (clean cutover) because the
  only consumer is the app's invite screen, updated in the same change.
- Audit recording is best-effort: a failed audit write never blocks the
  audited operation (consistent with today's login audit).
- Persian (fa) remains the single locale; all new messages are Persian.

## Dependencies

- Existing first-run setup flow and its once-only lock.
- Existing invite lifecycle (issue/redeem) and its phone binding.
- Existing per-building manager check (a caller must be a current manager of
  the building) for object-level authorization of all new grant operations.
- Existing best-effort audit recording.
- Adding the superadmin to the account-role set widens the set of roles the
  system recognizes; how that lands in storage is a planning-phase decision.
