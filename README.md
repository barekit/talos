# Product Requirements Document (PRD): Talos

| Attribute | Details |
| :--- | :--- |
| **Project Name** | Talos |
| **Repository** | `github.com/barekit/talos` |
| **Vision** | To be the "Gin/Echo" of AI Agents in Go—lightweight, fast, and batteries-included. |
| **Status** | Pre-Alpha (Phase 1) |
| **Target Audience** | Go backend engineers, AI engineers moving to production, Platform engineers. |

---

## 1. Executive Summary
Talos is an open-source framework for building autonomous AI agents in Golang. While Python dominates the AI prototyping space (LangChain, Agno), Go lacks a framework that balances **simplicity** with **production readiness**.

Talos fills this gap by providing an "Agno-like" experience: flexible tool usage (via reflection), built-in memory management, and easy RAG integration, all while leveraging Go's concurrency and type safety for high-performance production workloads.

## 2. Problem Statement
Building AI agents in Go currently requires choosing between:
1.  **Raw SDKs:** Manually handling chat history loops, JSON schema generation for tools, and error handling (Too low-level).
2.  **LangChainGo:** A port of the Python library that is often overly verbose, complex, and heavy (Too high-level/complex).

There is no "middle ground" framework that allows a developer to spin up an agent with memory and tools in under 50 lines of code.

## 3. Core Values (The "Talos Philosophy")
* **Idiomatic Go:** No forcing Python patterns into Go. We use interfaces, structs, and options patterns.
* **Batteries Included:** Memory (SQL), Knowledge (Vector), and structured logging come out of the box.
* **Reflection Magic:** We handle the tedious JSON Schema generation for OpenAI functions so the user doesn't have to.
* **Zero Dependency Bloat:** Keep the core module lean.

## 4. Feature Specifications

### 4.1. Phase 1: The Core Agent (MVP)
*Goal: Can we make an agent that uses a calculator tool?*

* **Tool Reflection Engine:**
    * Convert `func Add(a int, b int) int` -> OpenAI JSON Schema automatically.
    * Handle basic types (`int`, `string`, `bool`, `float`).
* **LLM Provider Interface:**
    * Generic `Chat()` interface.
    * **OpenAI Provider:** Implementation using the official `github.com/openai/openai-go`.
* **Agent Loop:**
    * Basic `Think -> Act -> Observe` loop.
    * Auto-execution of tool calls returned by the LLM.

### 4.2. Phase 2: Memory & Persistence
*Goal: The agent remembers who I am across restarts.*

* **Storage Interface:**
    * `Save(sessionID, message)` and `Load(sessionID)`.
* **Implementations:**
    * `In-Memory` (for testing).
    * `SQLite` (via `mattn/go-sqlite3` or CGO-free alternatives).
    * `Postgres` (via `pgx`).

### 4.3. Phase 3: Knowledge (RAG)
*Goal: The agent can read a PDF and answer questions about it.*

* **Knowledge Base:**
    * Simple API: `agent.AddKnowledge("company_policy.pdf")`.
* **Components:**
    * **Embedder:** OpenAI `text-embedding-3-small`.
    * **Vector DB:** Simple local vector store (or integration with Qdrant/Pinecone).
    * **Reranker:** Basic support for re-ranking results (optional for Alpha).

### 4.4. Phase 4: Multi-Agent Orchestration (Swarm)
*Goal: A "Manager" agent delegates tasks to a "Coder" agent.*

* **Handoffs:** Mechanism for one agent to call another agent as a tool.
* **Structured Output:** Enforce agents to return JSON matching a Go struct.

## 5. Technical Architecture

### Directory Structure
```text
/pkg
  /agent      # Orchestration logic
  /llm        # Model adapters (OpenAI, Anthropic)
  /tools      # Reflection & execution logic
  /memory     # Chat history persistence
  /knowledge  # RAG pipelines