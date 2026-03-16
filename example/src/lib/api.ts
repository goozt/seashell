import type {
  AdminStats,
  ApiResponse,
  ArchiveMeta,
  ArchivedBlock,
  AuthData,
  Authority,
  AuthorityStats,
  Block,
  FieldDefinition,
  Invitation,
  KYCRecord,
  KYCSummary,
  NetworkNode,
  NodeJoinRequest,
  SuperAdminStats,
  Ticket,
  TokenPair,
  Transaction,
  User,
  ValueRecord,
  VerificationConfig,
  VerificationStatusResponse,
  VerificationSubmission,
  WalletInfo,
} from "@/types/api";

// ---------------------------------------------------------------------------
// Token storage helpers (browser-safe)
// ---------------------------------------------------------------------------

const ACCESS_KEY = "seashell_access";
const REFRESH_KEY = "seashell_refresh";

export const tokenStore = {
  getAccess: () =>
    typeof window !== "undefined" ? localStorage.getItem(ACCESS_KEY) : null,
  getRefresh: () =>
    typeof window !== "undefined" ? localStorage.getItem(REFRESH_KEY) : null,
  set: (tokens: TokenPair) => {
    localStorage.setItem(ACCESS_KEY, tokens.access_token);
    localStorage.setItem(REFRESH_KEY, tokens.refresh_token);
  },
  clear: () => {
    localStorage.removeItem(ACCESS_KEY);
    localStorage.removeItem(REFRESH_KEY);
  },
};

// ---------------------------------------------------------------------------
// Base fetch helper
// ---------------------------------------------------------------------------

class ApiError extends Error {
  constructor(
    public status: number,
    message: string
  ) {
    super(message);
    this.name = "ApiError";
  }
}

async function request<T>(
  path: string,
  options: RequestInit = {},
  authenticated = true
): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };

  if (authenticated) {
    const token = tokenStore.getAccess();
    if (token) headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`/api/v1${path}`, {
    ...options,
    headers,
  });

  // Auto-refresh on 401
  if (res.status === 401 && authenticated) {
    const refreshed = await tryRefresh();
    if (refreshed) {
      headers["Authorization"] = `Bearer ${tokenStore.getAccess()}`;
      const retried = await fetch(`/api/v1${path}`, { ...options, headers });
      if (retried.ok) {
        const json: ApiResponse<T> = await retried.json();
        if (json.success && json.data !== undefined) return json.data;
      }
    }
    tokenStore.clear();
    if (typeof window !== "undefined") {
      window.dispatchEvent(new Event("seashell:session-expired"));
    }
    throw new ApiError(401, "Session expired. Please log in again.");
  }

  if (res.status === 204) return undefined as T;

  const json: ApiResponse<T> = await res.json();

  if (!res.ok || !json.success) {
    throw new ApiError(res.status, json.error ?? "Request failed");
  }

  return (json.data ?? null) as T;
}

async function tryRefresh(): Promise<boolean> {
  const refresh_token = tokenStore.getRefresh();
  if (!refresh_token) return false;
  try {
    const res = await fetch("/api/v1/auth/refresh", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token }),
    });
    if (!res.ok) return false;
    const json: ApiResponse<TokenPair> = await res.json();
    if (json.success && json.data) {
      tokenStore.set(json.data);
      return true;
    }
  } catch {
    /* swallow */
  }
  return false;
}

// ---------------------------------------------------------------------------
// Public
// ---------------------------------------------------------------------------

export const publicApi = {
  health: () => request<{ status: string; service: string }>("/health", {}, false),
  getBlocks: () => request<Block[]>("/blocks", {}, false),
  getBlock: (hash: string) => request<Block>(`/blocks/${hash}`, {}, false),
  getValue: () => request<ValueRecord[]>("/value", {}, false),
};

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

export const authApi = {
  register: (body: { username: string; email: string; password: string }) =>
    request<AuthData>("/auth/register", {
      method: "POST",
      body: JSON.stringify(body),
    }, false),

  login: (body: { email: string; password: string }) =>
    request<AuthData>("/auth/login", {
      method: "POST",
      body: JSON.stringify(body),
    }, false),

  logout: async () => {
    const refresh_token = tokenStore.getRefresh();
    try {
      await request("/auth/logout", {
        method: "POST",
        body: JSON.stringify({ refresh_token }),
      }, false);
    } finally {
      tokenStore.clear();
    }
  },

  me: () => request<User>("/auth/me"),

  refresh: async (): Promise<User | null> => {
    const ok = await tryRefresh();
    if (!ok) return null;
    try {
      return await request<User>("/auth/me");
    } catch {
      return null;
    }
  },
};

// ---------------------------------------------------------------------------
// User
// ---------------------------------------------------------------------------

export const userApi = {
  getProfile: () => request<User>("/user/me"),

  updateProfile: (body: { username?: string; email?: string; password?: string }) =>
    request<User>("/user/me", { method: "PUT", body: JSON.stringify(body) }),

  getWallet: () => request<WalletInfo>("/user/wallet"),

  createWallet: () =>
    request<WalletInfo>("/user/wallet", { method: "POST" }),

  getTransactions: () => request<Transaction[]>("/user/transactions"),

  sendTransaction: (body: { to_address: string; amount: number }) =>
    request<Transaction>("/user/transactions", {
      method: "POST",
      body: JSON.stringify(body),
    }),

  getAuthority: () => request<Authority>("/user/authority"),

  createAuthority: (body: { name: string; description: string }) =>
    request<Authority>("/user/authority", {
      method: "POST",
      body: JSON.stringify(body),
    }),

  joinAuthority: (body: { code: string }) =>
    request<{ authority: Authority; authority_role: string }>(
      "/user/authority/join",
      { method: "POST", body: JSON.stringify(body) }
    ),

  getTickets: () => request<Ticket[]>("/user/tickets"),

  createTicket: (body: { title: string; description?: string }) =>
    request<Ticket>("/user/tickets", {
      method: "POST",
      body: JSON.stringify(body),
    }),

  getTicket: (id: string) => request<Ticket>(`/user/tickets/${id}`),

  replyTicket: (id: string, body: { message: string }) =>
    request<Ticket>(`/user/tickets/${id}/reply`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
};

// ---------------------------------------------------------------------------
// Authority Owner
// ---------------------------------------------------------------------------

export const authorityApi = {
  getMembers: () => request<User[]>("/authority/members"),

  removeMember: (userId: string) =>
    request<{ message: string }>(`/authority/members/${userId}`, {
      method: "DELETE",
    }),

  getInvitations: () => request<Invitation[]>("/authority/invitations"),

  createInvitation: () =>
    request<Invitation>("/authority/invitations", { method: "POST" }),

  revokeInvitation: (code: string) =>
    request<void>(`/authority/invitations/${code}`, { method: "DELETE" }),

  getTransactions: () => request<Transaction[]>("/authority/transactions"),

  getValue: () => request<ValueRecord[]>("/authority/value"),

  getStats: () => request<AuthorityStats>("/authority/stats"),
};

// ---------------------------------------------------------------------------
// Admin
// ---------------------------------------------------------------------------

export const adminApi = {
  getRequests: () => request<Authority[]>("/admin/authority-requests"),

  getRequest: (id: string) => request<Authority>(`/admin/authority-requests/${id}`),

  approveRequest: (id: string, body: { base_price: number; sensitivity_k: number }) =>
    request<Authority>(`/admin/authority-requests/${id}/approve`, {
      method: "POST",
      body: JSON.stringify(body),
    }),

  rejectRequest: (id: string, body: { reason?: string }) =>
    request<Authority>(`/admin/authority-requests/${id}/reject`, {
      method: "POST",
      body: JSON.stringify(body),
    }),

  getTickets: () => request<Ticket[]>("/admin/tickets"),

  getTicket: (id: string) => request<Ticket>(`/admin/tickets/${id}`),

  updateTicket: (id: string, body: { status: string }) =>
    request<Ticket>(`/admin/tickets/${id}`, {
      method: "PUT",
      body: JSON.stringify(body),
    }),

  replyTicket: (id: string, body: { message: string }) =>
    request<Ticket>(`/admin/tickets/${id}/reply`, {
      method: "POST",
      body: JSON.stringify(body),
    }),

  // Escalate a node-level ticket to regional admin
  escalateToRegional: (id: string, note?: string) =>
    request<Ticket>(`/admin/tickets/${id}/escalate`, {
      method: "POST",
      body: JSON.stringify({ note: note ?? "" }),
    }),

  // Get tickets escalated to regional level
  getRegionalTickets: () =>
    request<Ticket[]>("/admin/tickets/regional"),

  // Escalate a regional ticket to superadmin
  escalateToSuper: (id: string, note?: string) =>
    request<Ticket>(`/admin/tickets/${id}/escalate-super`, {
      method: "POST",
      body: JSON.stringify({ note: note ?? "" }),
    }),

  getAuthorities: () => request<Authority[]>("/admin/authorities"),

  getUsers: () => request<User[]>("/admin/users"),

  getStats: () => request<AdminStats>("/admin/stats"),
};

// ---------------------------------------------------------------------------
// SuperAdmin
// ---------------------------------------------------------------------------

export const superAdminApi = {
  getAdmins: () => request<User[]>("/superadmin/admins"),

  createAdmin: (body: { username: string; email: string; password: string }) =>
    request<User>("/superadmin/admins", {
      method: "POST",
      body: JSON.stringify(body),
    }),

  demoteAdmin: (id: string) =>
    request<{ message: string }>(`/superadmin/admins/${id}`, {
      method: "DELETE",
    }),

  suspendAuthority: (id: string) =>
    request<Authority>(`/superadmin/authorities/${id}/suspend`, {
      method: "POST",
    }),

  reinstateAuthority: (id: string) =>
    request<Authority>(`/superadmin/authorities/${id}/reinstate`, {
      method: "POST",
    }),

  getStats: () => request<SuperAdminStats>("/superadmin/stats"),

  // Tickets escalated to superadmin
  getTickets: () => request<Ticket[]>("/superadmin/tickets"),
  replyTicket: (id: string, body: { message: string }) =>
    request<Ticket>(`/superadmin/tickets/${id}/reply`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
  updateTicket: (id: string, body: { status: string }) =>
    request<Ticket>(`/superadmin/tickets/${id}`, {
      method: "PUT",
      body: JSON.stringify(body),
    }),
};

// ---------------------------------------------------------------------------
// Node / Network API
// ---------------------------------------------------------------------------

export const nodeApi = {
  // Admin: list all nodes
  getNodes: () =>
    request<{ nodes: NetworkNode[] }>("/admin/nodes"),

  // Admin: get single node
  getNode: (id: string) =>
    request<NetworkNode>(`/admin/nodes/${id}`),

  // SuperAdmin (primary only): pending node join requests
  getNodeRequests: () =>
    request<{ requests: NodeJoinRequest[] }>("/superadmin/node-requests"),

  // SuperAdmin: approve a node join request
  approveNodeRequest: (id: string) =>
    request<{ node_id: string; peers: NetworkNode[] }>(
      `/superadmin/node-requests/${id}/approve`,
      { method: "POST" }
    ),

  // SuperAdmin: reject a node join request
  rejectNodeRequest: (id: string, reason?: string) =>
    request<{ message: string }>(`/superadmin/node-requests/${id}/reject`, {
      method: "POST",
      body: JSON.stringify({ reason: reason ?? "" }),
    }),

  // SuperAdmin: suspend a node
  suspendNode: (id: string) =>
    request<NetworkNode>(`/superadmin/nodes/${id}/suspend`, { method: "POST" }),

  // SuperAdmin: reinstate a node
  reinstateNode: (id: string) =>
    request<NetworkNode>(`/superadmin/nodes/${id}/reinstate`, { method: "POST" }),

  // Admin: list nodes by tier
  getNodesByTier: (tier: string) =>
    request<{ nodes: NetworkNode[]; tier: string }>(`/admin/nodes/tier/${tier}`),
};

// ---------------------------------------------------------------------------
// KYC
// ---------------------------------------------------------------------------

export const kycApi = {
  // User: initiate KYC verification
  initiateKYC: (body: { national_id: string; tax_id?: string }) =>
    request<KYCRecord>("/user/kyc", { method: "POST", body: JSON.stringify(body) }),

  // User: get own KYC status
  getMyKYC: () => request<KYCRecord>("/user/kyc"),

  // Admin: list all KYC records (optionally filter by status)
  listKYC: (status?: string) =>
    request<{ records: KYCRecord[] }>(`/admin/kyc${status ? `?status=${status}` : ""}`),

  // Admin: reject a KYC record
  rejectKYC: (userID: string, reason: string) =>
    request<KYCRecord>(`/admin/kyc/${userID}/reject`, {
      method: "POST",
      body: JSON.stringify({ reason }),
    }),

  // Admin: KYC summary for this node
  getSummary: () => request<KYCSummary>("/admin/kyc/summary"),
};

// ---------------------------------------------------------------------------
// Verification
// ---------------------------------------------------------------------------

export const verificationApi = {
  // User: get verification status + config + submission
  getMyVerification: () =>
    request<VerificationStatusResponse>("/user/verification"),

  // User: submit verification form
  submitVerification: (body: { field_values: Record<string, string> }) =>
    request<VerificationSubmission>("/user/verification", {
      method: "POST",
      body: JSON.stringify(body),
    }),

  // Authority owner: save/update verification config
  saveConfig: (body: { method: string; fields: FieldDefinition[] }) =>
    request<VerificationConfig>("/authority/verification/config", {
      method: "PUT",
      body: JSON.stringify(body),
    }),

  // Authority owner: get current config
  getConfig: () =>
    request<VerificationConfig>("/authority/verification/config"),

  // Authority owner: list submissions
  listSubmissions: (status?: string) =>
    request<VerificationSubmission[]>(
      `/authority/verification/submissions${status ? `?status=${status}` : ""}`
    ),

  // Authority owner: get single submission
  getSubmission: (id: string) =>
    request<VerificationSubmission>(`/authority/verification/submissions/${id}`),

  // Authority owner: approve/reject
  reviewSubmission: (id: string, body: { action: "approve" | "reject"; remarks?: string }) =>
    request<VerificationSubmission>(
      `/authority/verification/submissions/${id}/review`,
      { method: "POST", body: JSON.stringify(body) }
    ),
};

// ---------------------------------------------------------------------------
// Archive
// ---------------------------------------------------------------------------

export const archiveApi = {
  // Admin: archive statistics
  getStats: () => request<ArchiveMeta>("/admin/archive/stats"),

  // Admin: fetch a single archived block by height
  getBlock: (height: number) =>
    request<ArchivedBlock>(`/admin/archive/blocks/${height}`),

  // Admin: fetch a range of archived blocks
  getRange: (from: number, to: number) =>
    request<{ blocks: ArchivedBlock[]; total: number }>(
      `/admin/archive/blocks?from=${from}&to=${to}`
    ),
};
