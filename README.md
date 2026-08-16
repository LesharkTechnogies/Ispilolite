# Ispilo Lite — Backend API

<div align="center">

## 🌐 Ispilo Lite

**Connecting ISPs • Technicians • Customers**

[![Go](https://img.shields.io/badge/Backend-Go-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Status](https://img.shields.io/badge/Status-Under%20Development-orange)](#)
[![Platform](https://img.shields.io/badge/Platform-Kenya-green)](#)
[![Flutter App](https://img.shields.io/badge/Mobile-Flutter-02569B?logo=flutter&logoColor=white)](#)

### 👥 One platform. Three sides. Better connectivity.

**Customers** → Discover ISP coverage → Request installation → Get matched

**ISPs** → Receive customer requests → Provide quotations → Assign technicians

**Technicians** → Find installation jobs → Get assigned → Complete verified work

<br>

![Visitors](https://komarev.com/ghpvc/?username=IspiloKenya&label=Repository%20Visitors&color=0e75b6&style=for-the-badge)

</div>

---

## 🚧 Project Status

> **Under Active Development**
>
> Ispilo Lite is a work in progress. APIs, endpoints, architecture, and features may change as development continues.
>
> **Not yet production-ready.**

## 🔧 Backend API

This repository contains the **Ispilo Lite backend API**, written in **Go**.

The API provides the foundation for:

- 🔐 Authentication and user management
- 📍 ISP coverage and geospatial discovery
- 👷 Technician discovery and matching
- 📋 Installation requests and jobs
- 💰 Quotations and provider matching
- ⭐ Reviews and ratings
- 📡 API services for the future mobile application

The current backend architecture is organized around services for **authentication, core operations, geospatial functionality, and matching**.

## 📱 Mobile Application

The Ispilo Lite mobile application will be developed separately using **Flutter**.

```text
                 ISPILO LITE
                     │
          ┌──────────┼──────────┐
          │          │          │
       👤 Customer  🌐 ISP   👷 Technician
          │          │          │
          └──────────┼──────────┘
                     │
                Go Backend API
                     │
          ┌──────────┼──────────┐
          │          │          │
        Auth      Geo/Maps   Matching
                     │
                 Database
```

## 🏗️ Backend Services

| Service | Purpose |
|---|---|
| 🔐 Auth | Authentication, OTP & user management |
| ⚙️ Core | Profiles, users, reviews & CRUD operations |
| 📍 Geospatial | Locations, ISP coverage & nearby search |
| 🤝 Matching | Installation requests, quotations & job matching |

The API documentation defines the service ports as **8001–8004** and uses the `/api/v1/...` API versioning structure.

## 🎯 Vision

Ispilo Lite is being built to make it easier for people in Kenya to:

> **Find available internet providers, request installations, and connect with qualified technicians — from one platform.**

---

## 👨‍💻 Prepared by Leshark Technologies

Built with ❤️ for the Kenyan connectivity ecosystem.

💬 **Have an idea, suggestion, or want to collaborate?**

**WhatsApp:** [wa.me/254794052875](https://wa.me/254794052875)

**WhatsApp:** `IspiloKenya`

---

<div align="center">

### 🇰🇪 Built for Kenya • 🚀 Under Development • 🐹 Powered by Go

**Ispilo Lite**

</div>
