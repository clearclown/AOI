# Frontend Player Task

## Mission
You are the Frontend Player. Your mission is to create a **working React + TypeScript frontend** for the AOI UI following **TDD** principles.

## Current Context
- Project: AOI (Agent Operational Interconnect)
- Working Directory: /home/ablaze/Projects/AOI
- Specifications:
  - `/home/ablaze/Projects/AOI/.aida/specs/aoi-protocol-requirements.md`
  - `/home/ablaze/Projects/AOI/.aida/specs/aoi-protocol-design.md`

## Your Deliverables

### 1. Initialize Project

```bash
cd /home/ablaze/Projects/AOI
npm create vite@latest frontend -- --template react-ts
cd frontend
npm install
npm install -D vitest @testing-library/react @testing-library/jest-dom jsdom
npm install -D @vitest/ui
```

### 2. Directory Structure

```
frontend/
├── src/
│   ├── components/
│   │   ├── ApprovalUI/
│   │   │   ├── ApprovalUI.tsx
│   │   │   └── ApprovalUI.test.tsx
│   │   ├── AuditLog/
│   │   │   ├── AuditLog.tsx
│   │   │   └── AuditLog.test.tsx
│   │   └── Dashboard/
│   │       ├── Dashboard.tsx
│   │       └── Dashboard.test.tsx
│   ├── hooks/
│   │   └── useApi.ts
│   ├── services/
│   │   └── api.ts
│   ├── types/
│   │   └── index.ts
│   ├── App.tsx
│   ├── App.test.tsx
│   └── main.tsx
├── public/
├── index.html
├── package.json
├── vite.config.ts
├── vitest.config.ts
├── tsconfig.json
└── README.md
```

### 3. Core Components (MVP - Keep it Simple!)

#### A. Dashboard Component
**Purpose**: Main status display

**Features**:
- Display agent list
- Show connection status
- Simple card layout

**Test First**:
```tsx
// Dashboard.test.tsx
import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import Dashboard from './Dashboard';

describe('Dashboard', () => {
  it('renders dashboard title', () => {
    render(<Dashboard />);
    expect(screen.getByText(/AOI Dashboard/i)).toBeInTheDocument();
  });

  it('displays agent list', () => {
    render(<Dashboard />);
    expect(screen.getByText(/Agents/i)).toBeInTheDocument();
  });
});
```

**Then Implement**:
```tsx
// Dashboard.tsx
export default function Dashboard() {
  return (
    <div className="dashboard">
      <h1>AOI Dashboard</h1>
      <section>
        <h2>Agents</h2>
        <p>No agents connected</p>
      </section>
    </div>
  );
}
```

#### B. ApprovalUI Component
**Purpose**: Human-in-the-loop approval interface

**Features**:
- Display pending approval requests
- Approve/Deny buttons
- Timeout countdown

**Test First**:
```tsx
// ApprovalUI.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import ApprovalUI from './ApprovalUI';

describe('ApprovalUI', () => {
  it('renders approval title', () => {
    render(<ApprovalUI />);
    expect(screen.getByText(/Approval Requests/i)).toBeInTheDocument();
  });

  it('shows no pending requests message', () => {
    render(<ApprovalUI />);
    expect(screen.getByText(/No pending requests/i)).toBeInTheDocument();
  });
});
```

#### C. AuditLog Component
**Purpose**: Timeline of agent communications

**Features**:
- Display audit events chronologically
- Show event details
- Search/filter (basic)

**Test First**:
```tsx
// AuditLog.test.tsx
import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import AuditLog from './AuditLog';

describe('AuditLog', () => {
  it('renders audit log title', () => {
    render(<AuditLog />);
    expect(screen.getByText(/Audit Log/i)).toBeInTheDocument();
  });

  it('shows empty state', () => {
    render(<AuditLog />);
    expect(screen.getByText(/No events/i)).toBeInTheDocument();
  });
});
```

### 4. Configuration Files

#### vite.config.ts
```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
```

#### vitest.config.ts
```typescript
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
  },
})
```

#### src/test/setup.ts
```typescript
import { expect, afterEach } from 'vitest';
import { cleanup } from '@testing-library/react';
import * as matchers from '@testing-library/jest-dom/matchers';

expect.extend(matchers);

afterEach(() => {
  cleanup();
});
```

### 5. TDD Protocol

For EACH component:
1. **RED**: Write test first (it fails)
2. **GREEN**: Implement component (test passes)
3. **REFACTOR**: Clean up code

### 6. API Service (Stub for MVP)

```typescript
// src/services/api.ts
export interface Agent {
  id: string;
  role: string;
  status: 'online' | 'offline';
}

export interface AuditEvent {
  id: string;
  timestamp: string;
  type: string;
  summary: string;
}

export interface ApprovalRequest {
  id: string;
  requester: string;
  taskType: string;
  status: 'pending' | 'approved' | 'denied';
}

export const api = {
  getAgents: async (): Promise<Agent[]> => {
    // Mock data for MVP
    return [];
  },

  getAuditLog: async (): Promise<AuditEvent[]> => {
    return [];
  },

  getApprovalRequests: async (): Promise<ApprovalRequest[]> => {
    return [];
  },

  approveRequest: async (id: string): Promise<void> => {
    console.log('Approved:', id);
  },

  denyRequest: async (id: string): Promise<void> => {
    console.log('Denied:', id);
  },
};
```

### 7. Main App

```tsx
// src/App.tsx
import Dashboard from './components/Dashboard/Dashboard';
import ApprovalUI from './components/ApprovalUI/ApprovalUI';
import AuditLog from './components/AuditLog/AuditLog';
import './App.css';

function App() {
  return (
    <div className="app">
      <header>
        <h1>AOI Protocol UI</h1>
      </header>
      <main>
        <Dashboard />
        <ApprovalUI />
        <AuditLog />
      </main>
    </div>
  );
}

export default App;
```

### 8. Quality Gates

Before you declare completion, verify:
1. ✅ `npm install` succeeds
2. ✅ `npm run build` succeeds
3. ✅ `npm test -- --run` passes (minimum 3 test files)
4. ✅ `npm run dev` starts dev server
5. ✅ Browser shows UI at localhost:3000
6. ✅ All components render without errors

### 9. File Checklist

Minimum files to create:
1. `frontend/package.json` - Dependencies
2. `frontend/vite.config.ts` - Vite config
3. `frontend/vitest.config.ts` - Test config
4. `frontend/src/test/setup.ts` - Test setup
5. `frontend/src/types/index.ts` - TypeScript types
6. `frontend/src/services/api.ts` - API service
7. `frontend/src/components/Dashboard/Dashboard.tsx`
8. `frontend/src/components/Dashboard/Dashboard.test.tsx`
9. `frontend/src/components/ApprovalUI/ApprovalUI.tsx`
10. `frontend/src/components/ApprovalUI/ApprovalUI.test.tsx`
11. `frontend/src/components/AuditLog/AuditLog.tsx`
12. `frontend/src/components/AuditLog/AuditLog.test.tsx`
13. `frontend/src/App.tsx`
14. `frontend/src/App.test.tsx`
15. `frontend/README.md`

### 10. Minimum Viable Implementation

**DO implement**:
- ✅ Basic component structure
- ✅ Mock data for display
- ✅ Component tests
- ✅ TypeScript types
- ✅ Clean UI layout

**DO NOT implement** (save for later):
- ❌ Real API integration (use mocks)
- ❌ WebSocket connections
- ❌ Complex state management
- ❌ Advanced animations
- ❌ Responsive design (desktop only for MVP)

### 11. Example Test Run

```bash
cd /home/ablaze/Projects/AOI/frontend
npm test -- --run

# Should output:
# ✓ src/components/Dashboard/Dashboard.test.tsx (2 tests)
# ✓ src/components/ApprovalUI/ApprovalUI.test.tsx (2 tests)
# ✓ src/components/AuditLog/AuditLog.test.tsx (2 tests)
#
# Test Files  3 passed (3)
# Tests  6 passed (6)
```

## Success Criteria

You are DONE when:
- [ ] All components created with tests
- [ ] Minimum 3 test files (*.test.tsx)
- [ ] `npm run build` succeeds
- [ ] `npm test -- --run` passes
- [ ] UI renders without errors
- [ ] README.md documents how to run

## Notes
- Keep styling minimal (basic CSS)
- Use TypeScript strict mode
- All props must be typed
- Mock API calls for MVP
- Focus on structure, not polish

## Start Here
1. Read the specs first
2. Initialize Vite project
3. Install dependencies
4. Follow TDD: Test → Component → Test
5. Verify quality gates
6. Report completion

Good luck! 🎨
