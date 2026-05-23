// =======================================================
// REQUEST INTERFACES
// =======================================================

import { UserProfile } from "@/types/user.types";

export interface RegisterRequest {
    full_name: string;
    email: string;
    password: string;
}

export interface VerifyEmailRequest {
    token: string;
}

export interface ResendEmailRequest {
    email: string;
}

export interface LoginRequest {
    email: string;
    password: string;
}

export interface ForgotPasswordRequest extends ResendEmailRequest {}

export interface ResetPasswordRequest {
    token: string;
    new_password: string;
}

export interface Enable2FARequest {
    code: string
}

export interface Verify2FALoginRequest {
    code: string;
    method: "totp" | "backup_code";
}

// =======================================================
// RESPONSE INTERFACES
// =======================================================

export interface AuthResponse {
    access_token: string;
    user: UserProfile
}

export interface MessageReponse {
    message: string;
}

export interface Setup2FAResponse {
    secret: string;
    qr_code_uri: string;
    backup_codes: string[];
}

export interface TwoFAStateResponse {
    is_enabled: boolean;
    message: string;
    backup_remaining: number;
}
