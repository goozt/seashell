// ── Shared ────────────────────────────────────────────────────────────────────

export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
}

// ── Auth ──────────────────────────────────────────────────────────────────────

export interface User {
  id: string;
  username: string;
  email: string;
  role: "user" | "admin" | "superadmin";
  authority_id?: string;
  authority_role?: "owner" | "member";
  wallet_address?: string;
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface AuthData {
  user: User;
  tokens: TokenPair;
}

// ── Blockchain ────────────────────────────────────────────────────────────────

export interface Block {
  hash: string;
  height: number;
  timestamp: string;
  valid_poa: boolean;
  tx_count: number;
}

export interface ValueRecord {
  authority_id: string;
  authority_name: string;
  price: number;
  block_height: number;
  recorded_at?: string;
  transaction_volume?: number;
  circulating_supply?: number;
  velocity?: number;
}

// ── Wallet ────────────────────────────────────────────────────────────────────

export interface WalletInfo {
  address: string;
  balance: number;
  message?: string;
}

// ── Transaction ───────────────────────────────────────────────────────────────

export interface Transaction {
  id?: string;
  from_address?: string;
  to_address?: string;
  amount?: number;
  block_height?: number;
  timestamp?: string;
  member_id?: string;
  member_username?: string;
}

// ── Authority ─────────────────────────────────────────────────────────────────

export interface Authority {
  id: string;
  name: string;
  description?: string;
  owner_id: string;
  status: "pending" | "active" | "suspended" | "rejected";
  base_price: number;
  sensitivity_k: number;
  validator_pubkey?: string;
  approved_at?: string;
  approved_by?: string;
  rejection_reason?: string;
  created_at: string;
}

export interface AuthorityStats {
  member_count: number;
  ticket_count: number;
  current_price: number;
}

// ── Invitation ────────────────────────────────────────────────────────────────

export interface Invitation {
  code: string;
  authority_id: string;
  created_by: string;
  used_by?: string;
  expires_at: string;
  used: boolean;
  created_at: string;
}

// ── Tickets ───────────────────────────────────────────────────────────────────

export interface TicketReply {
  author_id: string;
  author_username?: string;
  message: string;
  created_at: string;
}

export interface Ticket {
  id: string;
  authority_id: string;
  created_by: string;
  title: string;
  description?: string;
  status: "open" | "in_progress" | "resolved" | "closed";
  replies: TicketReply[];
  created_at: string;
  updated_at: string;
}

// ── Admin Stats ───────────────────────────────────────────────────────────────

export interface AdminStats {
  total_users: number;
  total_authorities: number;
  total_tickets: number;
}

export interface SuperAdminStats extends AdminStats {
  total_admins: number;
  active_authorities: number;
  pending_authorities: number;
}
