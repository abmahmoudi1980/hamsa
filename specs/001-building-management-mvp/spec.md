# Feature Specification: Building Management MVP (P0)

**Feature Branch**: `001-building-management-mvp`

**Created**: 2026-08-26

**Status**: Draft

**Input**: User description: "MVP/P0 requirements document for a building management software (Hamsa) covering charge calculation and billing, payments and debt tracking, expenses, maintenance requests, announcements, resident panel, manager dashboard, and authentication — as specified in `mvp-p0.md`."

## User Scenarios & Testing *(mandatory)*

<!--
  User stories are ordered as prioritized user journeys. Each is independently testable:
  implementing only the P1 stories already delivers a viable product (charge calculation
  and collection). P2/P3 stories extend it toward the full P0 scope.
-->

### User Story 1 - Authenticate and control access (Priority: P1)

A manager or a resident logs in with their mobile number and a password. The first account on a fresh deployment is created via setup bootstrap as the platform superadmin (002-multi-manager-support); managers are created by superadmin- or manager-issued **manager-role** invites, and residents register with a one-time invite code (default role), which also serves as the password-recovery path. A building can have any number of managers — an existing manager grants and revokes co-managers per building. After login, the system determines who they are: a manager sees the buildings they manage, and a resident sees only the data of the unit(s) they are linked to. A resident can never see financial data of other residents, and a manager...

**Why this priority**: Access control is the gate to every other story; without it, the financial data of residents cannot be safely exposed. It is also a hard security requirement of the product.

**Independent Test**: A manager and a resident account can be created, both can log in, and the resident's view is verifiably restricted to their own unit's data.

**Acceptance Scenarios**:

1. **Given** a registered mobile number, **When** the user logs in with the correct password, **Then** the user gains access; new residents join via a one-time invite code from their manager.
2. **Given** a logged-in resident, **When** they browse the app, **Then** only data of their own unit (invoices, payments, requests, announcements) is visible.
3. **Given** a logged-in manager, **When** they browse the app, **Then** only buildings they manage are accessible.
4. **Given** a sensitive action such as issuing an invoice, **When** the action completes, **Then** the audit log contains the acting user, the operation, the timestamp, and the affected object (with before/after values for sensitive changes).

---

### User Story 2 - Set up building, floors, and units (Priority: P1)

A manager registers a building (name, address, number of blocks/floors/units, construction year, manager contact, emergency contact, notes) and then defines its units. Each unit records its number, block, floor, area, parking spaces, storage rooms, number of occupants, and status (active, vacant, occupied, inactive). The system prevents duplicate unit numbers within a building, allows editing, keeps a change history, and supports searching and filtering units.

**Why this priority**: The building and unit registry is the foundation for every financial calculation; no invoice can exist without units.

**Independent Test**: A manager can create a building with multiple units, and duplicate unit numbers are rejected — the registry is complete and correct before any billing is configured.

**Acceptance Scenarios**:

1. **Given** a manager account, **When** they create a building and add units, **Then** each unit is stored with all its attributes and appears in searchable, filterable lists.
2. **Given** a building with unit 5 already defined, **When** the manager tries to create another unit 5 in the same building, **Then** the system rejects the duplicate.
3. **Given** an existing unit, **When** the manager edits its details, **Then** the change is saved and the previous state is retained in the change history.

---

### User Story 3 - Manage owners and residents with occupancy history (Priority: P1)

A manager links people (owners, tenants, non-resident owners) to units. Each person record includes full name, mobile number, national ID when required, relationship to the unit, occupancy start/end dates, and active/inactive status. A unit may have one or more owners and one or more residents. When a resident changes, the previous record is preserved rather than deleted, and the number of occupants per unit is tracked as its own data point — changeable, with history — because it feeds the charge calculation.

**Why this priority**: Occupancy and occupant counts are direct inputs to the charge engine (per-person method) and to resident access; they must exist before billing.

**Independent Test**: A manager can register a resident for a unit, later replace them with a new resident, and verify the old occupancy history and occupant count history remain intact.

**Acceptance Scenarios**:

1. **Given** a unit, **When** the manager adds an owner and a tenant with occupancy dates, **Then** both relationships are stored against the unit.
2. **Given** a unit with an existing resident, **When** the manager records a new resident, **Then** the previous resident's record becomes inactive but their occupancy history is preserved.
3. **Given** a unit, **When** the manager changes the number of occupants, **Then** the new value is stored along with its history and the previous invoices are not automatically altered.

---

### User Story 4 - Define charge formula, calculate, review, and issue invoices (Priority: P1)

A manager defines a billing period (month, start date, end date, due date, status: draft → calculated → issued → closed) and adds one or more cost items to it. Each cost item uses a calculation method: equal split among units, per-occupant, per-area, fixed amount, or specific-units-only (e.g., parking, in-unit repair, damages), and multiple methods can be combined in one invoice (e.g., 50% equal + 30% area + 20% occupants). The system calculates each unit's share, applies prior balance, late fees, and credits, and shows the result for review before issuing. Late fees can be none, fixed, percentage, or per-day/per-period. Once issued, an invoice never changes automatically when base data (occupant count, area) changes; corrections require an explicit recalculate/correct operation. Cancelling an invoice preserves its history.

**Why this priority**: This is the core of the product — the single most important outcome of P0 is that the manager can correctly calculate and collect the charge.

**Independent Test**: Using the sample scenario (10 units, 10,000,000 equal-split general cost, 8,000,000 per-occupant water cost, 20 total occupants, unit 1 has 4 occupants), the system must produce invoice of 2,600,000 for unit 1, reviewable before issue.

**Acceptance Scenarios**:

1. **Given** a building with 10 units and cost items as in the sample scenario, **When** the manager calculates the period, **Then** unit 1's invoice equals 2,600,000 (1,000,000 + 1,600,000) and each invoice lists its cost line items.
2. **Given** a calculated but not issued period, **When** the manager reviews the results, **Then** the calculation method and each unit's share are visible and verifiable before issuance.
3. **Given** an issued invoice, **When** the occupant count of the unit is later changed, **Then** the issued invoice amount does not change.
4. **Given** an issued invoice that needs correction, **When** the manager performs an explicit correction/recalculation, **Then** the change happens as a recorded operation, not silently.
5. **Given** rounding of shares (e.g., equal split of an amount not divisible by unit count), **When** the calculation completes, **Then** the sum of all unit shares reconciles exactly with the total cost.
6. **Given** a period with prior debt, late fee, and credit for a unit, **When** the invoice is issued, **Then** the final amount = prior debt + current charge + late fee − payments − credits, with the late fee shown as a separate line item.
7. **Given** an issued invoice, **When** the manager cancels it, **Then** the invoice is marked cancelled and its history is retained (not deleted).

---

### User Story 5 - Record payments and track unit balances (Priority: P2)

A manager records manual payments (amount, date, related invoice, method, tracking number, recorded-by), and residents can pay online through a payment gateway. Partial payments are supported (a 3,000,000 invoice paid with 1,500,000 leaves 1,500,000 outstanding). The system maintains for every unit: prior debt, current invoice, late fee, credits, payments, and remaining balance, and shows invoice payment status (unpaid, partially paid, paid, expired, cancelled).

**Why this priority**: Collection is the second half of the core financial loop; without payment recording there is no balance or debt visibility.

**Independent Test**: A manager can record a partial manual payment against an invoice and the unit's balance and invoice status update correctly (paid → partially paid with correct remainder).

**Acceptance Scenarios**:

1. **Given** an issued invoice of 3,000,000, **When** the manager records a payment of 1,500,000, **Then** the invoice status becomes "partially paid" and the unit's remaining balance reflects 1,500,000 outstanding.
2. **Given** a resident with an outstanding invoice, **When** they pay online successfully, **Then** the payment is recorded with a tracking number and the invoice status updates accordingly.
3. **Given** any unit, **When** the manager views its financial status, **Then** the system shows prior debt, current charge, late fee, payments, credits, and the resulting balance.

---

### User Story 6 - Register building expenses and view basic financial report (Priority: P2)

A manager records building expenses with title, category (water, electricity, gas, elevator, cleaning, security, repairs, insurance, equipment, other), amount, date, description, payer, receipt image/attachment, and approval status. The system provides a basic financial report showing monthly income, monthly expense, net income/expense, total resident debt, total payments received, and total expenses.

**Why this priority**: Expense tracking justifies the charge amounts to residents and enables the manager to verify that collected charges cover costs.

**Independent Test**: After recording several categorized expenses, the financial report totals match the sum of recorded expenses and collected payments.

**Acceptance Scenarios**:

1. **Given** the manager role, **When** they register an expense with a receipt image, **Then** the expense is stored with all attributes and appears under its category.
2. **Given** a month with recorded expenses and payments, **When** the manager opens the financial report, **Then** monthly income, monthly expense, net figure, total resident debt, total payments, and total expenses are all displayed and consistent with the underlying records.

---

### User Story 7 - Submit and manage maintenance requests (Priority: P3)

A resident submits a maintenance request with title, category (elevator, utilities, electrical, plumbing/water, cleaning, common areas, parking, other), description, location, photo, submission date, and priority (normal, important, urgent). The request moves through statuses: new → under review → in progress → done → closed. The manager can assign a responsible person, add notes, record costs, and change status or close the request. The resident can track the status of their own request at any time.

**Why this priority**: Maintenance is a daily pain point in building management, but it does not block the financial core; it completes the "operations" pillar.

**Independent Test**: A resident submits a request, the manager processes it through its statuses attaching a cost, and the resident sees every status change without contacting the manager.

**Acceptance Scenarios**:

1. **Given** a logged-in resident, **When** they submit a maintenance request with photo and priority, **Then** the request is recorded with date and appears as "new" to the manager.
2. **Given** a new request, **When** the manager assigns a responsible person and moves it to "in progress" and later "done", **Then** the resident sees each status change and the recorded cost.
3. **Given** a completed request, **When** the manager closes it, **Then** the request reaches final state "closed" and remains in history.

---

### User Story 8 - Publish announcements and generate notifications (Priority: P3)

A manager publishes announcements with title, text, publish date, expiry date, audience (whole building, a block, a floor, or a specific unit), and optional attachment. Announcements are visible inside the system; push notification delivery is supported when the infrastructure allows. Key events also generate notifications: invoice issued, due date approaching, payment overdue, payment recorded, maintenance request submitted, maintenance status changed, announcement published.

**Why this priority**: Communication removes the need for phone calls and message groups, but it is an amplifier of the core flows rather than a prerequisite.

**Independent Test**: A manager publishes an announcement targeted at one block, and only residents of that block see it; issuing an invoice produces a notification for affected residents.

**Acceptance Scenarios**:

1. **Given** a manager, **When** they publish an announcement targeting block B, **Then** only residents of block B see the announcement, within its publish/expiry window.
2. **Given** an issued invoice for a unit, **When** issuance completes, **Then** the unit's resident receives a notification.
3. **Given** an unpaid invoice past its due date, **When** the due date passes, **Then** an overdue notification is generated.

---

### User Story 9 - Resident self-service panel (Priority: P2)

A resident logs in and sees their home screen: financial status (amount payable, latest invoice, due date, payment status), open maintenance requests (count, latest request and its status), and announcements (latest items, unread count). From the main menu — home, charges, payments, maintenance, announcements, profile — they can view invoices, view debt and payment history, pay a bill and see the receipt, and track their requests, all without contacting the manager.

**Why this priority**: Resident self-service is one of the three defining outcomes of P0 ("residents track their financial status and requests without contacting the manager").

**Independent Test**: A resident with one issued invoice and one open request can log in and answer "what do I owe, by when, and what happened to my request?" entirely from the panel.

**Acceptance Scenarios**:

1. **Given** a resident with an issued invoice, **When** they open the home screen, **Then** payable amount, latest invoice, due date, and payment status are displayed.
2. **Given** a resident with an open maintenance request, **When** they view the requests section, **Then** the request and its current status are shown.
3. **Given** new announcements targeting the resident's unit, **When** they open the announcements section, **Then** they see the latest items and the count of unread ones.
4. **Given** an outstanding invoice, **When** the resident completes payment, **Then** a receipt is viewable and the payable amount updates.

---

### User Story 10 - Manager dashboard with alerts (Priority: P3)

A manager opens the dashboard and sees the building's status at a glance: unit count, debtor units, total debt amount, monthly income, monthly expense, and open maintenance requests, plus quick actions (issue charge, record expense, record payment, record request, send announcement) and alerts such as "7 units are in debt", "invoices of 5 units are past due", "3 maintenance requests are open", "an expense is awaiting confirmation".

**Why this priority**: The dashboard makes daily management fast once the underlying flows exist, but it adds no new capability on its own.

**Independent Test**: Given known data (e.g., 7 debtor units and 3 open requests), the dashboard cards and alerts display exactly those numbers.

**Acceptance Scenarios**:

1. **Given** a building with 7 debtor units and 3 open maintenance requests, **When** the manager opens the dashboard, **Then** the cards and alerts show these counts and the total debt amount.
2. **Given** the dashboard, **When** the manager uses a quick action such as "record expense", **Then** they are taken directly to that operation.

---

### Edge Cases

- What happens when an equal-split cost is not divisible by the number of participating units (rounding residue)? → The system must reconcile the sum of unit shares with the cost total and handle rounding differences explicitly (BR-09).
- What happens when a participating factor is zero (e.g., all units have zero occupants for the per-occupant method, or a specific-unit cost item with no units selected)? → The system must block or warn, not divide by zero.
- What happens when the occupant count or area of a unit changes after invoices were issued? → Issued invoices must remain unchanged (BR-05); only explicit correction operations alter them.
- What happens when a payment exceeds the remaining balance of an invoice? → The surplus is treated as a credit for the unit.
- What happens when a vacant unit exists at billing time? → Whether a vacant unit is included or excluded depends on each cost item's inclusion rule (BR-06, BR-07).
- What happens when a resident linked to a unit is deactivated or replaced mid-period? → Occupancy history is preserved and only units linked at calculation time (per the cost item rules) are considered.
- What happens when a period is closed and a late payment arrives? → The payment is still recorded against the unit's balance history.
- What happens when a user has multiple roles (e.g., resident in one unit, manager of another building)? → Access is the union of permitted scopes, with resident data isolation still enforced.
- What happens when an announcement's expiry date passes? → It stops being shown to residents but remains in history.

## Requirements *(mandatory)*

### Functional Requirements

**Setup & Registry**

- **FR-001**: System MUST allow a manager to register a building with name, address, block/floor/unit counts, construction year, manager contact, emergency contact, and notes.
- **FR-002**: System MUST allow a manager to create, edit, search, and filter units with number, block, floor, area, parking count/number(s), storage count/number(s), occupant count, and status (active, vacant, occupied, inactive).
- **FR-003**: System MUST reject duplicate unit numbers within the same building.
- **FR-004**: System MUST retain change history for important unit and building data changes.

**People & Occupancy**

- **FR-005**: System MUST allow linking people to units as owner, tenant, or non-resident owner, with full name, mobile number, national ID (when used), relationship, occupancy start/end dates, and active/inactive status.
- **FR-006**: System MUST support multiple owners and multiple residents per unit.
- **FR-007**: System MUST replace a resident without deleting prior occupancy records, preserving full occupancy history.
- **FR-008**: System MUST store the occupant count of each unit as an independent, editable data point with its own history, usable as an input to charge calculation.

**Charge Engine & Invoicing**

- **FR-009**: System MUST allow a manager to define billing periods with month, start date, end date, due date, and status (draft, calculated, issued, closed).
- **FR-010**: System MUST allow multiple cost items per billing period, each with a title, total amount, and a calculation method.
- **FR-011**: System MUST support calculation methods: (A) equal split among participating units, (B) proportional to unit occupant count, (C) proportional to unit area, (D) fixed amount per unit, and (E) specific-units-only costs (e.g., parking, in-unit repair, damages).
- **FR-012**: System MUST support combining multiple methods within a single invoice (e.g., percentage mixtures such as 50% equal + 30% area + 20% occupants).
- **FR-013**: System MUST let each cost item define which units are included, including the option to include or exclude vacant units.
- **FR-014**: System MUST show the calculation method and per-unit shares for review before invoices are issued (BR-08).
- **FR-015**: System MUST compute each unit's invoice as: prior debt + current charge + late fee − payments − credits, displaying prior debt, late fee, and credits as separate items.
- **FR-016**: System MUST support simple late-fee definitions: none, fixed amount, percentage of amount, or fixed amount per day/period, displayed as a separate invoice item.
- **FR-017**: System MUST NOT alter issued invoices when base data (occupant count, area) changes; corrections MUST occur only through an explicit recalculate/correct operation (BR-03, BR-05).
- **FR-018**: System MUST reconcile rounding so that the sum of unit shares equals the cost item total (BR-09).
- **FR-019**: System MUST assign each invoice a number, unit, period, base amount, cost items, prior debt, late fee, credits, final amount, issue date, due date, and payment status (unpaid, partially paid, paid, expired, cancelled).
- **FR-020**: System MUST preserve the history of a cancelled invoice instead of deleting it (BR-10).
- **FR-021**: System MUST calculate invoices for a typical building within seconds.

**Payments & Balances**

- **FR-022**: System MUST support manual payment recording by the manager with amount, date, related invoice, payment method, tracking number, and recording user.
- **FR-023**: System MUST support online payment through a payment gateway for residents.
- **FR-024**: System MUST support partial payments, updating invoice status and unit balance accordingly (BR-02).
- **FR-025**: System MUST maintain per-unit current balance components (prior debt, current invoice, late fee, credits, payments, remaining balance), where each unit has exactly one current financial status but unlimited invoice history (BR-01).

**Expenses & Reporting**

- **FR-026**: System MUST allow a manager to record expenses with title, category, amount, date, description, payer, receipt attachment, and approval status, with predefined categories (water, electricity, gas, elevator, cleaning, security, repairs, insurance, equipment, other).
- **FR-027**: System MUST provide a basic financial report showing monthly income, monthly expense, net income/expense, total resident debt, total payments received, and total expenses.

**Maintenance**

- **FR-028**: System MUST allow residents to submit maintenance requests with title, category, description, location, photo, date, and priority (normal, important, urgent), with categories (elevator, utilities, electrical, water, cleaning, common areas, parking, other).
- **FR-029**: System MUST move requests through statuses new → under review → in progress → done → closed, and allow the manager to assign a responsible person, add notes, record costs, change status, and close requests.
- **FR-030**: System MUST allow residents to view the current status of their own requests.

**Announcements & Notifications**

- **FR-031**: System MUST allow a manager to publish announcements with title, text, publish date, expiry date, audience (whole building, block, floor, specific unit), and optional attachment.
- **FR-032**: System MUST display announcements inside the system to the targeted audience, and support push notification delivery when infrastructure permits.
- **FR-033**: System MUST generate notifications for: invoice issued, due date approaching, payment overdue, payment recorded, maintenance request submitted, maintenance status changed, announcement published.

**Panels & Dashboard**

- **FR-034**: System MUST provide a resident panel home screen showing payable amount, latest invoice, due date, payment status, open request count and latest request status, latest announcements, and unread announcement count, with a main menu of home, charges, payments, maintenance, announcements, and profile.
- **FR-035**: System MUST provide a manager dashboard showing unit count, debtor units, total debt, monthly income, monthly expense, open requests, quick actions (issue charge, record expense, record payment, record request, send announcement), and alerts for debtor units, past-due invoices, open requests, and expenses awaiting confirmation.

**Authentication, Access & Audit**

- **FR-036**: System MUST authenticate users via mobile number + password, support first-account setup bootstrap (creating the single superadmin) and role-carrying one-time invite codes issued by the superadmin or managers (registration + password recovery; redemption never changes an existing user's role), and route users to their accessible buildings/units after login (updated by specs/002-multi-manager-support).
- **FR-037**: System MUST enforce that residents see only their own unit's data, that managers see only their permitted buildings, and that one resident's financial data is never visible to another resident.
- **FR-038**: System MUST record an audit log with user, operation, timestamp, and affected object for: unit create/edit, resident change, charge formula change, invoice issue, invoice cancel, payment recording, expense recording, and maintenance status change — including before/after values for sensitive operations.

### Key Entities *(include if feature involves data)*

- **Building**: A managed property with name, address, block/floor/unit counts, construction year, manager contact, emergency contact, notes.
- **Unit**: A dwelling within a building with number (unique per building), block, floor, area, parking, storage, occupant count, status.
- **Person**: An owner, tenant, or non-resident owner identified by name, mobile number, optional national ID.
- **Occupancy**: The dated, historical link between a person and a unit (start/end dates, active flag), preserving history across resident changes.
- **OccupantCountHistory**: Time-stamped record of each unit's occupant count, usable as a charge-calculation input.
- **BillingPeriod**: A charge cycle with month, start/end/due dates and lifecycle status (draft, calculated, issued, closed).
- **CostItem**: A line item of a period with title, total amount, calculation method (equal, per-occupant, per-area, fixed, specific-units, or a combination), and unit-inclusion rules.
- **Invoice**: The calculated bill for one unit in one period, with number, line items, prior debt, late fee, credits, final amount, issue/due dates, and payment status; immutable after issue except via explicit correction.
- **Payment**: A recorded transaction (manual or online) with amount, date, related invoice, method, tracking number, recorder; may be partial.
- **Expense**: A building cost with title, category, amount, date, description, payer, receipt attachment, approval status.
- **UnitBalance**: The unit's current financial position (prior debt, current charge, late fee, credits, payments, remaining balance).
- **MaintenanceRequest**: A resident-submitted issue with title, category, description, location, photo, priority, status, assignee, notes, and costs.
- **Announcement**: A published notice with title, text, publish/expiry dates, audience scope, and optional attachment.
- **Notification**: An event-generated message to a user (invoice issued, due date approaching, overdue, payment recorded, maintenance events, announcement published).
- **AuditLog**: A record of user, operation, timestamp, affected object, and before/after values for sensitive operations.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A manager can complete the full setup-to-first-invoice flow (create building, define units and residents, define charge formula, create a period, calculate, review, issue) without any technical-team involvement, per the MVP acceptance list.
- **SC-002**: For a typical building (up to ~50 units) with multiple cost items and mixed calculation methods, the system produces all invoices within seconds, and 100% of calculated shares reconcile with cost totals including rounding.
- **SC-003**: The sample scenario (10 units, equal 10,000,000 + per-occupant 8,000,000 water, unit 1 with 4 of 20 occupants) yields exactly 2,600,000 for unit 1 — verifiable before issuance.
- **SC-004**: 100% of issued invoices remain unchanged when base data (occupant count, area) is subsequently modified; changes occur only through explicit correction operations.
- **SC-005**: Residents can determine their payable amount, due date, and payment status, and complete a payment, entirely on their own — measured by a resident completing view-invoice → pay → view-receipt without contacting the manager.
- **SC-006**: A resident can track the status of a submitted maintenance request through all status changes without contacting the manager.
- **SC-007**: Managers can answer "who owes what?" in one dashboard view: debtor unit count, total debt, monthly income, and monthly expense are visible at a glance.
- **SC-008**: No resident can ever access another resident's financial data (verifiable by access-control testing across all resident-facing screens).
- **SC-009**: All sensitive operations (invoice issue/cancel, payment, expense, formula change, status changes) are traceable in the audit log with user, time, and object, with 100% coverage of the listed operation types.
- **SC-010**: Financial and transaction records survive simple deletion of a user or unit — data is deactivated/archived rather than destroyed.

## Assumptions

- Currency is the Iranian Toman (amounts shown as in the requirements document); amounts are stored in whole units without fractions beyond rounding rules.
- Dates follow the Persian (Jalali) calendar for user-facing display.
- One manager may manage one or more buildings; a resident may be linked to one or more units; a person can hold different roles across buildings.
- Online payment via a payment gateway is in P0 scope, but the specific gateway provider is an implementation decision deferred to planning.
- Push notification delivery is best-effort in P0 — in-system notifications and announcements are the guaranteed channel.
- The system is used in a single locale (Persian) with mobile-friendly web access; a separate native mobile app is out of scope for P0.
- Multiple buildings per installation are supported at the data level, but P0 usage focuses on a manager operating one or a few buildings.
- Items explicitly excluded from P0 (and therefore from this spec): advanced parking management, common-area reservation, contractor contract management, full double-entry accounting, inventory, access control/entry, plate recognition, IoT, AI, multi-layer board management, standalone native app, CRM, and advanced financial reports.
- Notification channels beyond in-app (SMS/email) are optional and infrastructure-dependent; their absence does not block any acceptance scenario.
- The resident login identity is the mobile number; a non-resident owner uses the same resident account type with a "non-resident owner" relationship and the same restricted visibility.
