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
  first_name: string;
  last_name: string;
  email: string;
  role: "user" | "admin" | "superadmin";
  authority_id?: string;
  authority_role?: "owner" | "member";
  wallet_address?: string;
  active: boolean;
  created_at: string;
  updated_at: string;
  // KYC fields
  kyc_status?: KYCStatus;
  national_id_masked?: string;
  tax_id_masked?: string;
  verified_name?: string;
  kyc_verified_at?: string;
  kyc_rejected_reason?: string;
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
  author_role?: string;
  message: string;
  created_at: string;
}

export type TicketLevel = "node" | "regional" | "superadmin";

export interface Ticket {
  id: string;
  authority_id: string;
  created_by: string;
  title: string;
  description?: string;
  status: "open" | "in_progress" | "resolved" | "closed";
  escalated_to?: TicketLevel;
  escalation_note?: string;
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

// ── Network Nodes ─────────────────────────────────────────────────────────────

export type NodeStatus = "pending" | "active" | "offline" | "suspended" | "rejected";

export interface NetworkNode {
  id: string;
  authority_id?: string;
  authority_name: string;
  node_url: string;
  validator_pubkey?: string;
  status: NodeStatus;
  is_primary: boolean;
  block_height: number;
  version?: string;
  last_seen_at?: string;
  registered_at: string;
  approved_at?: string;
  approved_by?: string;
  rejection_reason?: string;
  // 3-tier topology fields
  node_tier?: "primary" | "regional" | "branch";
  parent_node_id?: string;
  parent_node_url?: string;
  cert_fingerprint?: string;
}

export interface NodeJoinRequest {
  id: string;
  node_url: string;
  authority_name: string;
  admin_email: string;
  validator_pubkey: string;
  status: "pending" | "approved" | "rejected";
  created_at: string;
  reviewed_at?: string;
  reviewed_by?: string;
  rejection_reason?: string;
}

export interface NetworkStats {
  total: number;
  active: number;
  offline: number;
  pending: number;
}

// ── KYC ───────────────────────────────────────────────────────────────────────

export type KYCStatus = "unverified" | "pending" | "verified" | "rejected";

export interface KYCRecord {
  user_id: string;
  national_id_masked: string;
  tax_id_masked?: string;
  status: KYCStatus;
  verified_name?: string;
  verified_at?: string;
  rejected_reason?: string;
  node_id?: string;
  created_at: string;
  updated_at: string;
}

export interface KYCSummary {
  node_id: string;
  total: number;
  verified: number;
  pending: number;
  rejected: number;
  unverified: number;
}

// ── Verification ─────────────────────────────────────────────────────────────

export type VerificationMethod = "manual";
export type VerifySubmissionStatus = "pending" | "approved" | "rejected";

export interface FieldDefinition {
  name: string;
  label: string;
  input_type: "text" | "number" | "email" | "textarea" | "select";
  required: boolean;
  pattern?: string;
  min_length?: number;
  max_length?: number;
  options?: string[];
  placeholder?: string;
}

export interface VerificationConfig {
  authority_id: string;
  method: VerificationMethod;
  fields: FieldDefinition[];
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface VerificationSubmission {
  id: string;
  authority_id: string;
  user_id: string;
  method: string;
  field_values: Record<string, string>;
  status: VerifySubmissionStatus;
  reviewed_by?: string;
  reviewed_at?: string;
  remarks?: string;
  username?: string;
  full_name?: string;
  created_at: string;
  updated_at: string;
}

export interface VerificationStatusResponse {
  config_status: "not_configured" | "configured";
  config?: VerificationConfig;
  submission?: VerificationSubmission;
}

// ── Quorum ────────────────────────────────────────────────────────────────────

export interface ValidatorSig {
  pub_key: string;
  sig: string;
}

// ── Archive ───────────────────────────────────────────────────────────────────

export interface ArchiveMeta {
  oldest_height: number;
  newest_height: number;
  total_blocks: number;
  updated_at: string;
}

export interface ArchivedBlock {
  hash: string;
  prev_hash: string;
  height: number;
  timestamp: number;
  signatures: ValidatorSig[];
}
