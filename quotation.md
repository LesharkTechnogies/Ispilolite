# ISPilo Quotation System
## Requirements & Implementation Specification

## 1. Overview

ISPilo is a quotation-generation platform designed primarily for **ISPs and technology businesses**, while keeping the quotation engine generic enough to support other businesses in the future.

The system will use:

- **Go** for the backend, business logic, database access, quotation finalization, public quotation links, and PDF generation services where required.
- **Flutter** for the application UI, quotation creation, editing, preview, offline quotation generation, sharing, and downloading.
- **PostgreSQL** for persistent storage.
- **Flutter local storage/database** for offline quotation creation and preview.

The most important design principle is:

> **A quotation being created is not automatically stored in the server database. It only becomes a permanent quotation when the user downloads, shares, or otherwise finalizes it.**

---

# 2. Business Types

ISPilo must support different business types.

Initial supported types:

```text
ISP
TECHNOLOGY
OTHER
```

### ISP businesses

Examples:

- Internet subscriptions
- Internet installation
- Router installation
- Network installation
- Fiber installation
- Wireless links
- Bandwidth packages
- Network equipment
- Maintenance

### Technology businesses

Examples:

- IT support
- Computer repair
- Hardware sales
- Software installation
- Network configuration
- Cybersecurity services
- Consultancy
- Software services
- Technical support

The quotation engine must remain generic so more business categories can be added later.

---

# 3. Company Profile

Each company should have a profile containing:

```text
companies
-------------------------
id
name
business_type
phone
email
address
website
tax_pin
logo_url
logo_enabled
primary_color
secondary_color
created_at
updated_at
```

### Logo rules

For ISPs:

- Company logo should be supported and prominently displayed.
- Logo should appear **centered at the top of the quotation**.

For technology businesses:

- Company logo is optional.
- If a logo is not provided, the quotation should remain professional without an empty logo area.

### Document hierarchy

The PDF should visually follow:

```text
Company Logo
      ↓
Company Name
      ↓
Business Description / Contacts
      ↓
QUOTATION
      ↓
Customer
      ↓
Items
      ↓
Financial Summary
      ↓
Payment Information
      ↓
Terms
      ↓
ISPilo Branding
      ↓
Disclaimer
```

---

# 4. Quotation Lifecycle

A quotation should have two stages.

## Stage 1 — Creation

While the user is creating a quotation:

- Data exists in Flutter memory/local storage.
- No permanent server quotation is created.
- User can add, edit, remove, and preview items.
- User can preview the final PDF offline.
- Custom units can be used even when there is no internet connection.

```text
Create quotation
      ↓
Flutter
      ↓
Local quotation state
      ↓
PDF Preview
```

## Stage 2 — Finalization

When the user chooses an action such as:

- Download
- Share
- Generate public link

the quotation becomes a finalized quotation.

```text
Download / Share / Generate Link
              ↓
        Finalize quotation
              ↓
       Generate unique ID
              ↓
        Save quotation
              ↓
      Generate final document
```

This prevents abandoned or incomplete quotations from unnecessarily filling the server database.

---

# 5. Quotation Number

Each finalized quotation should have a human-readable quotation number.

Recommended format:

```text
QT-YYYY-MM-SEQUENCE
```

Example:

```text
QT-2026-08-00124
```

Meaning:

```text
QT       = quotation
2026     = year
08       = month
00124    = quotation sequence
```

The quotation number is for business/reference purposes.

It should be different from the public quotation URL identifier.

---

# 6. Public Quotation Identifier

Every finalized quotation should receive a unique public code.

Format:

```text
ILO + random alphanumeric characters
```

Length:

- Minimum: **8 characters**
- Maximum: **16 characters**
- Recommended default: **12 characters**

Examples:

```text
ILO7K4M92
ILO8FQ2X7
ILO7K4M92X8P
ILO4X8P2Q7Z91
```

Public URL:

```text
https://quotations.ispilo.co.ke/ILO7K4M92X8P
```

The public code should be:

- Automatically generated.
- Unique.
- Difficult to guess.
- Case-consistent, preferably uppercase.
- Independent from the database primary key.

Do not expose the internal UUID as the public quotation identifier.

---

# 7. Public Code Generation

The generation principle should be similar to transaction/reference code generation systems such as M-Pesa codes, but **must not copy M-Pesa's algorithm**.

Use a cryptographically secure random generator.

Recommended alphabet:

```text
ABCDEFGHJKLMNPQRSTUVWXYZ23456789
```

Ambiguous characters such as:

```text
0
O
1
I
```

can be excluded.

Generation process:

```text
Generate random characters
        ↓
Add ILO prefix
        ↓
Check database uniqueness
        ↓
Already exists?
   YES       NO
    ↓         ↓
 Generate    Save
 again
```

Database constraint:

```text
UNIQUE(public_code)
```

Example:

```text
ILO7K4M92X8P
```

---

# 8. Quotation Database

## quotations

```text
quotations
-------------------------
id
public_code
company_id
customer_id

quotation_number

subtotal

discount_type
discount_value
discount_amount

transport_enabled
transport_amount

tax_enabled
taxable_amount
tax_amount
tax_rate
tax_mode

total_amount

status

created_at
updated_at
finalized_at
downloaded_at
expires_at
```

Possible statuses:

```text
DRAFT_LOCAL
FINALIZED
SENT
VIEWED
ACCEPTED
REJECTED
EXPIRED
```

However, `DRAFT_LOCAL` does not need to exist in the server database because draft quotations can remain entirely in Flutter until finalized.

---

# 9. Quotation Items

Each quotation contains multiple items.

## quotation_items

```text
quotation_items
-------------------------
id
quotation_id

item
description

unit_id
quantity
unit_price

discount_type
discount_value
discount_amount

amount

created_at
```

The quotation table should display:

| Item | Description | Qty | Unit Price | Discount | Amount |
|---|---|---:|---:|---:|---:|
| Cable Box | Cat6 UTP, 305m | 1 box | KSh 4,500 | KSh 500 | KSh 4,000 |
| Network Cable | Cat6 UTP | 20 meters | KSh 20 | KSh 0 | KSh 400 |

---

# 10. Item vs Description

`Item` and `Description` must be separate.

### Item

A short name:

```text
Cable Box
Network Cable
Router
Installation
Network Clips
```

### Description

Additional detail:

```text
Cat6 UTP, 305m
Cat6 UTP cable
MikroTik router
Complete router installation
5 pieces per pack
```

This keeps the quotation readable while allowing detailed descriptions.

---

# 11. Quantity and Units

Quantity must be stored as a **numeric value**.

The unit must be stored separately.

Do not store:

```text
"20 meters"
```

as the quantity.

Store:

```text
quantity = 20
unit_id = meter
```

The PDF can display:

```text
20 meters
```

but calculations must use:

```text
20
```

Therefore:

```text
20 meters × KSh 20
```

is calculated as:

```text
20 × 20 = KSh 400
```

Never attempt to calculate:

```text
"20 meters" × 20
```

---

# 12. Unit Database

Units should be database-driven.

## units

```text
units
-------------------------
id
name
singular_name
plural_name
symbol
is_system
company_id
created_at
```

Example:

| Name | Singular | Plural | Symbol |
|---|---|---|---|
| Meter | meter | meters | m |
| Piece | piece | pieces | pcs |
| Box | box | boxes | box |
| Roll | roll | rolls | roll |
| Pack | pack | packs | pkt |
| Set | set | sets | set |
| Kilogram | kilogram | kilograms | kg |
| Liter | liter | liters | L |
| Hour | hour | hours | hr |
| Day | day | days | day |
| Month | month | months | month |
| Service | service | services | svc |

---

# 13. System Units vs Custom Units

There should be two types of units.

## System units

ISPilo provides common units:

```text
is_system = true
```

Examples:

```text
meter
piece
box
pack
roll
set
kilogram
liter
hour
day
month
service
```

## Custom units

A company can create its own unit:

```text
is_system = false
company_id = company ID
```

Example:

```text
name = Melio
singular_name = melio
plural_name = melios
symbol = mel
```

Custom units belong to the company that created them.

This prevents one company's custom terminology from automatically appearing for every other ISPilo user.

---

# 14. Unit Autocomplete

When the user enters a unit, the system should provide autocomplete suggestions.

For example, if the user types:

```text
me
```

the application can display:

```text
Meter
Melio
```

If the user types:

```text
met
```

the system can show:

```text
Meter
```

Search should prioritize:

1. Exact matches.
2. Names starting with the entered text.
3. Names containing the entered text.

Example:

```text
User enters: me

Meter       ← starts with "me"
Melio       ← starts with "me"
Centimeter  ← contains "me"
```

---

# 15. Creating a Custom Unit

If no suitable unit exists, the user should be able to create one.

Example:

```text
User types:
megaliter

No matching unit

[ + Create "megaliter" ]
```

Then:

```text
Create Custom Unit

Name:
[ Megaliter ]

Symbol:
[ ML ]

[ Save ]
```

The new unit is saved against the company.

It should immediately become available for quotation creation.

---

# 16. Offline Custom Units

Offline functionality is important.

The Flutter application should maintain a local unit database/cache containing:

- ISPilo system units.
- The user's previously downloaded/synchronized custom units.

Therefore, a user can create a quotation without internet access.

Example:

```text
Flutter Offline
      ↓
Local Units DB
      ↓
Meter
Piece
Box
Melio
Pack
      ↓
Create quotation
      ↓
PDF preview
```

A custom unit created offline can be stored locally and synchronized with the Go backend when connectivity returns.

The quotation itself can also remain local until the user chooses to finalize/share/download it according to the application's finalization rules.

---

# 17. Pack / Bundle Quantities

The system must support items where the selling unit represents multiple physical items.

Example:

```text
5 pieces = KSh 1,000
```

The item configuration should contain:

```text
unit = Pack
pack_size = 5
unit_price = 1,000
```

If the customer requires 300 pieces:

```text
Required quantity = 300 pieces
Pack size = 5 pieces

300 / 5 = 60 packs

60 × 1,000 = KSh 60,000
```

The quotation can display:

```text
Network Clips
300 pieces (60 packs)
KSh 1,000 / pack
KSh 60,000
```

If partial packs are not allowed:

```text
packs = ceil(required_quantity / pack_size)
```

Example:

```text
23 pieces / 5

= 4.6
= 5 packs
```

Therefore:

```text
5 × 1,000 = KSh 5,000
```

The system should make it clear that 5 packs contain 25 pieces.

---

# 18. Unit Calculation Principle

Units are metadata for:

- Display.
- User understanding.
- Pack conversion.
- Quantity labeling.

They must not become part of the mathematical value.

Example:

```text
Displayed:
20 meters

Stored:
quantity = 20
unit = meter

Calculation:
20 × 20
```

Result:

```text
KSh 400
```

The same applies to:

```text
5 pieces
2 boxes
10 meters
3 rolls
4 sets
6 hours
2 days
```

---

# 19. Per-Item Discounts

Every quotation row should have its own discount.

Example:

| Item | Description | Qty | Unit Price | Discount | Amount |
|---|---|---:|---:|---:|---:|
| Cable Box | Cat6 UTP, 305m | 1 box | KSh 4,500 | KSh 500 | KSh 4,000 |

The discount can be:

```text
NONE
FIXED
PERCENTAGE
```

## Fixed discount

```text
Quantity × Unit Price = Gross Amount

Gross Amount - Discount = Final Amount
```

Example:

```text
1 × 4,500 = 4,500

4,500 - 500 = 4,000
```

## Percentage discount

```text
Gross Amount = Quantity × Unit Price

Discount Amount =
Gross Amount × Discount Percentage / 100

Final Amount =
Gross Amount - Discount Amount
```

Example:

```text
60 × 1,000 = 60,000

10% = 6,000

60,000 - 6,000 = 54,000
```

If there is no discount, the discount column can display:

```text
—
```

or:

```text
KSh 0
```

depending on the PDF design.

---

# 20. Transport Fee

Transport must be optional.

If transport is not entered:

```text
transport = 0
```

The PDF should not display an unnecessary transport line.

If transport is entered:

```text
Subtotal
- Item discounts
+ Transport
= Taxable amount
```

Example:

```text
Subtotal              KSh 20,000
Item Discounts       -KSh  2,000
Transport             KSh    500
────────────────────────────────
Taxable Amount        KSh 18,500
```

---

# 21. VAT / Tax

VAT must be optional.

The user should be able to choose:

```text
VAT applicable?
[ ] No
[✓] Yes
```

If VAT is disabled:

```text
VAT = 0
```

and the VAT line should not appear in the quotation.

If enabled, the user selects the configured tax rate.

Example:

```text
VAT rate: 16%
```

or another configured rate where applicable.

Do not hard-code one VAT rate into the application.

---

# 22. Tax Rate Database

## tax_rates

```text
tax_rates
-------------------------
id
name
rate
is_active
is_default
created_at
updated_at
```

Example:

| Name | Rate | Active | Default |
|---|---:|---|---|
| VAT | 16.00% | Yes | Yes |
| VAT | 3.00% | Yes | No |

The available rates should be configurable.

---

# 23. Quotation Tax Snapshot

When a quotation is finalized, save the actual tax details used on that quotation.

## quotation_tax

```text
quotation_tax
-------------------------
id
quotation_id
tax_rate_id
rate
calculation_type
taxable_amount
tax_amount
```

`calculation_type`:

```text
EXCLUSIVE
INCLUSIVE
```

The actual rate must be stored on the quotation tax record.

This is important because changing the tax configuration later must not alter historical quotations.

Example:

```text
Quotation QT-2026-08-00124
VAT rate at creation = 16%
VAT amount = KSh 1,680
```

Even if the configured rate changes later, the old quotation remains unchanged.

---

# 24. VAT Exclusive

If the quotation price is VAT exclusive:

```text
Taxable Amount = amount after discount + transport

VAT = Taxable Amount × VAT Rate / 100

Total = Taxable Amount + VAT
```

Example:

```text
Taxable Amount       KSh 10,500
VAT @ 16%            KSh  1,680
───────────────────────────────
TOTAL                KSh 12,180
```

---

# 25. VAT Inclusive

If the quoted total already includes VAT:

```text
VAT component =
Total × VAT Rate / (100 + VAT Rate)
```

Example with 16% VAT:

```text
Total = KSh 11,600

VAT =
11,600 × 16 / 116

= KSh 1,600
```

Therefore:

```text
Amount excluding VAT = KSh 10,000
VAT                  = KSh  1,600
Total                = KSh 11,600
```

---

# 26. Quotation Calculation Order

The calculation engine should use a consistent order.

For each item:

```text
Gross Item Amount
=
Quantity × Unit Price
```

Then:

```text
Item Amount
=
Gross Item Amount - Item Discount
```

After all items:

```text
Subtotal
=
Sum of Item Amounts
```

Then:

```text
Taxable Amount
=
Subtotal + Transport
```

Then apply VAT if enabled.

For VAT exclusive:

```text
VAT
=
Taxable Amount × Rate / 100

Total
=
Taxable Amount + VAT
```

For VAT inclusive:

```text
Total
=
Taxable Amount

VAT component
=
Total × Rate / (100 + Rate)
```

---

# 27. Payment Information

Payment information is optional but should be configurable at company level.

Supported methods:

```text
NONE
TILL
PAYBILL
BANK
CASH
OTHER
```

For M-Pesa Till:

```text
Payment Method: M-Pesa Till
Till Number: 5123456
```

For M-Pesa PayBill:

```text
Payment Method: M-Pesa PayBill
PayBill Number: 123456
Account Number: CUSTOMER-001
```

The quotation should only display fields relevant to the selected payment method.

---

# 28. Company Payment Settings

Payment details should normally be stored in the company profile rather than entered manually on every quotation.

Example:

```text
company_payment_methods
-------------------------
id
company_id
payment_method
till_number
paybill_number
account_number_format
bank_name
bank_account
is_default
is_active
```

The user selects the payment method while creating the quotation.

The application automatically loads the company's configured payment details.

---

# 29. Quotation PDF Layout

The company logo should be centered.

Example:

```text
┌─────────────────────────────────────────────────────────┐
│                                                         │
│                    [ COMPANY LOGO ]                     │
│                                                         │
│                    COMPANY NAME                         │
│              Internet & Technology Services             │
│                                                         │
│           Phone | Email | Website | PIN                 │
│                                                         │
├─────────────────────────────────────────────────────────┤
│                                                         │
│                       QUOTATION                          │
│                                                         │
│ Quotation No: QT-2026-08-00124                           │
│ Date:         14 August 2026                            │
│ Valid Until:  28 August 2026                            │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ BILL TO                                                 │
│                                                         │
│ Customer: John Doe                                      │
│ Phone:    0711 XXX XXX                                  │
│ Email:    john@example.com                              │
│ Location: Machakos                                      │
│                                                         │
├────┬──────────────┬────────────────┬─────┬──────────────┤
│ #  │ Item         │ Description    │ Qty │ Unit Price   │
├────┼──────────────┼────────────────┼─────┼──────────────┤
│ 1  │ Cable Box    │ Cat6 UTP 305m  │ 1   │ KSh 4,500    │
│ 2  │ Cable        │ Cat6 UTP       │ 20m │ KSh 20       │
└────┴──────────────┴────────────────┴─────┴──────────────┘
```

The full table should include:

```text
Item
Description
Qty
Unit Price
Discount
Amount
```

---

# 30. Footer

The ISPilo logo must always appear at the bottom of the quotation.

Example:

```text
                    [ ISPILO LOGO ]

                      ISPilo
          quotations.ispilo.co.ke/ILO7K4M92X8P

Disclaimer: ISPilo is not responsible for the contents
of this quotation; the issuing business is solely responsible.
```

The disclaimer should be small and unobtrusive.

---

# 31. ISPilo Watermark

Free quotations should contain a subtle ISPilo watermark.

The watermark can be:

```text
ISPilo
```

repeated diagonally or lightly across the document.

It should not interfere with:

- Company logo.
- Customer information.
- Item table.
- Totals.
- Payment information.

The footer ISPilo branding remains at the bottom.

---

# 32. Removing the Watermark

ISPilo can use watermark removal as a paid feature.

Example:

```text
Remove watermark
KSh 50
```

The payment must be verified by the backend before the watermark is removed.

Do not trust:

```text
Flutter:
paymentSuccessful = true
```

Instead:

```text
Flutter
   ↓
Payment request
   ↓
Payment provider
   ↓
Backend verification
   ↓
Payment confirmed
   ↓
Remove-watermark permission
   ↓
Generate PDF
```

---

# 33. Plans / Features

A plan system can control watermark behavior.

## plans

```text
plans
-------------------------
id
name
price
watermark_enabled
logo_enabled
max_quotations
created_at
```

Example:

```text
FREE
    Price: KSh 0
    Watermark: Yes

PREMIUM
    Price: KSh 50
    Watermark: No
```

Whether KSh 50 removes the watermark from one quotation or for a period should be a configurable product decision.

---

# 34. Payments

## payments

```text
payments
-------------------------
id
company_id
quotation_id
amount
currency
payment_method
transaction_reference
status
purpose
created_at
```

Example:

```text
amount = 50
currency = KES
payment_method = MPESA
purpose = REMOVE_WATERMARK
status = COMPLETED
```

---

# 35. Sharing

After a quotation is finalized, the user should have:

```text
[ Download PDF ]
[ Share ]
[ Copy Link ]
```

Share should generate a public URL:

```text
https://quotations.ispilo.co.ke/ILO7K4M92X8P
```

The URL can be shared through:

- WhatsApp
- SMS
- Email
- Telegram
- Copy Link
- Other applications supported by Flutter sharing

---

# 36. Public Quotation Page

When a client opens:

```text
https://quotations.ispilo.co.ke/ILO7K4M92X8P
```

they should see a web representation of the quotation.

Example:

```text
                 [ COMPANY LOGO ]

                  COMPANY NAME

                    QUOTATION

Quotation No: QT-2026-08-00124
Date: 14 August 2026

Customer:
John Doe

────────────────────────────────────────
Item              Qty    Amount
────────────────────────────────────────
Cable Box          1     KSh 4,000
Network Cable     20m    KSh   400
────────────────────────────────────────

Subtotal                 KSh 4,400
Transport                KSh   500
VAT @ 16%                KSh   784
────────────────────────────────────────
TOTAL                    KSh 5,684

          [ DOWNLOAD PDF ]

                 [ ISPILO LOGO ]
```

---

# 37. Flutter Offline Architecture

Flutter should contain a local quotation engine.

Suggested structure:

```text
Flutter
│
├── Presentation
│   ├── QuotationCreate
│   ├── QuotationPreview
│   ├── ItemEditor
│   ├── UnitAutocomplete
│   └── CompanySettings
│
├── Domain
│   ├── Quotation
│   ├── QuotationItem
│   ├── Unit
│   ├── TaxRate
│   └── CalculationEngine
│
├── Data
│   ├── LocalDatabase
│   ├── UnitRepository
│   └── QuotationRepository
│
└── PDF
    └── QuotationPdfRenderer
```

The calculation engine should work without an internet connection.

---

# 38. Offline Unit Storage

Flutter should have a local database/cache containing:

```text
System Units
+
Previously synchronized Company Custom Units
```

When online:

```text
Flutter
   ↓
Go API
   ↓
PostgreSQL
```

When offline:

```text
Flutter
   ↓
Local DB
```

This allows quotation creation and PDF preview without internet access.

---

# 39. Go Backend

Go should handle:

```text
Authentication
Company management
Customer management
Quotation finalization
Quotation persistence
Quotation number generation
Public code generation
Public quotation retrieval
Payment verification
Watermark permissions
Tax configuration
Unit synchronization
Quotation sharing
Audit records
```

Suggested architecture:

```text
Go
│
├── API
│
├── Authentication
│
├── Company Service
│
├── Customer Service
│
├── Quotation Service
│
├── Calculation/Validation Service
│
├── Unit Service
│
├── Tax Service
│
├── Payment Service
│
├── PDF Service
│
└── Public Quotation Service
```

---

# 40. Important Separation of Responsibilities

The system should distinguish between:

### Flutter

Responsible for:

- User interface.
- Local quotation creation.
- Offline calculation.
- Unit selection.
- Custom unit creation.
- PDF preview.
- Sharing UI.
- Download UI.

### Go

Responsible for:

- Authoritative server-side validation.
- Final quotation persistence.
- Unique quotation identifiers.
- Public links.
- Company settings.
- Tax configuration.
- Payment verification.
- Access control.
- Public quotation retrieval.

The backend should recalculate important financial values during finalization instead of blindly trusting values sent by Flutter.

---

# 41. Recommended Final Flow

```text
                 USER
                   │
                   ▼
          Create Quotation
                   │
                   ▼
               Flutter
                   │
        ┌──────────┴──────────┐
        │                     │
   Online                  Offline
        │                     │
        ▼                     ▼
 Server units             Local units
 Custom units             Custom units
        │                     │
        └──────────┬──────────┘
                   ▼
             Add Items
                   │
                   ▼
          Calculate locally
                   │
                   ▼
             PDF Preview
                   │
                   ▼
       ┌───────────┼───────────┐
       │           │           │
   Continue      Download     Share
   Editing          │           │
                    └─────┬─────┘
                          ▼
                   FINALIZE
                          │
                          ▼
                       Go API
                          │
                ┌─────────┴─────────┐
                │                   │
          Validate data       Generate ILO code
                │                   │
                └─────────┬─────────┘
                          ▼
                     PostgreSQL
                          │
              ┌───────────┴───────────┐
              ▼                       ▼
        Store quotation          Public URL
                                      │
                                      ▼
                 quotations.ispilo.co.ke/ILOXXXXXXXX
```

---

# 42. Core Design Principles

1. **Quotation creation should not automatically create a server record.**
2. **Finalized/downloaded/shared quotations must be stored.**
3. **Use UUIDs internally and ILO public codes externally.**
4. **Public codes must be unique and randomly generated.**
5. **Quotation numbers should show month and year.**
6. **Units must be stored separately from numeric quantities.**
7. **Units must support system and company-custom units.**
8. **Unit autocomplete should search as the user types.**
9. **Custom units must work offline.**
10. **Item and Description must be separate fields.**
11. **Each quotation row must have its own discount.**
12. **Discounts must support fixed and percentage values.**
13. **Transport must be optional.**
14. **VAT must be optional.**
15. **VAT rates must come from configurable database records.**
16. **VAT must support inclusive and exclusive calculation.**
17. **Historical quotations must retain the tax rate used when finalized.**
18. **Payment method must be optional and configurable.**
19. **Till payments should display the Till number.**
20. **PayBill payments should display PayBill and account number.**
21. **Company logos should be centered at the top.**
22. **Technology businesses must be able to generate quotations.**
23. **ISPilo branding should always appear at the bottom.**
24. **Free quotations may contain an ISPilo watermark.**
25. **Watermark removal should require verified payment.**
26. **The disclaimer should appear in one short sentence at the bottom.**
27. **Flutter must support offline quotation creation and PDF preview.**
28. **Go should be the authoritative backend for finalized quotations.**
29. **The server must recalculate and validate financial totals before finalization.**
30. **The same quotation data model should support both ISP and technology businesses.**

---

# 43. Example Final Quotation

```text
┌──────────────────────────────────────────────────────────────┐
│                                                              │
│                     [ COMPANY LOGO ]                         │
│                                                              │
│                    HOMENET SOLUTIONS                         │
│              Internet & Technology Services                 │
│          Nairobi | 0712 XXX XXX | info@company.co.ke         │
│                                                              │
├──────────────────────────────────────────────────────────────┤
│                         QUOTATION                             │
│                                                              │
│ QT-2026-08-00124                  14 August 2026             │
│ Valid Until: 28 August 2026                                  │
│                                                              │
│ BILL TO                                                      │
│ John Doe | 0711 XXX XXX | Machakos                          │
│                                                              │
├────┬──────────────┬────────────────┬──────┬────────┬─────────┤
│ #  │ Item         │ Description    │ Qty  │ Price  │Discount │
├────┼──────────────┼────────────────┼──────┼────────┼─────────┤
│ 1  │ Cable Box    │ Cat6 UTP 305m  │ 1box │ 4,500 │ 500     │
│ 2  │ Network Cable│ Cat6 UTP       │20m   │ 20     │ —       │
├────┴──────────────┴────────────────┴──────┴────────┴─────────┤
│                                                              │
│                                              Amount:         │
│                                         KSh 4,000            │
│                                         KSh   400            │
│                                                              │
│ Subtotal                                  KSh 4,400          │
│ Discount                                  KSh   500          │
│ Transport                                 KSh   500          │
│ Taxable Amount                            KSh 4,400          │
│ VAT @ 16%                                 KSh   704          │
│ ───────────────────────────────────────────────────────────  │
│ TOTAL                                     KSh 5,104          │
│                                                              │
├──────────────────────────────────────────────────────────────┤
│ PAYMENT INFORMATION                                          │
│                                                              │
│ M-Pesa PayBill                                               │
│ PayBill: 123456                                              │
│ Account: CUSTOMER-001                                        │
│                                                              │
├──────────────────────────────────────────────────────────────┤
│ TERMS & CONDITIONS                                           │
│                                                              │
│ Quotation valid for 14 days.                                 │
│ Prices and services are subject to the stated terms.         │
│                                                              │
│ Prepared By: __________________                               │
│                                                              │
│                  Thank you for your business!                 │
│                                                              │
│                       [ ISPILO LOGO ]                         │
│                           ISPilo                              │
│       quotations.ispilo.co.ke/ILO7K4M92X8P                    │
│                                                              │
│ Disclaimer: ISPilo is not responsible for the contents of    │
│ this quotation; the issuing business is solely responsible.  │
└──────────────────────────────────────────────────────────────┘
```

The PDF renderer should dynamically hide unused fields. For example, if there is no discount, transport, or VAT, those lines should not appear at all.

---

# 44. Final Architecture

```text
                         ISPILO
                           │
             ┌─────────────┴─────────────┐
             │                           │
          Flutter                        Go
             │                           │
       Quotation UI                REST API
       Offline DB                  PostgreSQL
       Unit search                Finalization
       Custom units               Public links
       Calculations               Payments
       PDF Preview                Validation
       Share/Download             Tax config
             │                           │
             └─────────────┬─────────────┘
                           │
                       PostgreSQL
                           │
               ┌───────────┴───────────┐
               │                       │
          Finalized                 Company
          Quotations                Settings
               │                       │
               ▼                       ▼
       Public quotation          Units / Taxes /
       ILOXXXXXXXX               Payments / Branding
```

The central principle is that **Flutter should be capable of producing a complete quotation preview and PDF offline using locally available data, while Go becomes authoritative when the quotation is finalized and made persistent/shareable online.**