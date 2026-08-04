import Constants from 'expo-constants';
import {
  getAccessToken, setAccessToken, getRefreshToken, saveSession, clearSession,
  type SessionUser,
} from './session';

// Em desenvolvimento o aparelho não enxerga "localhost" — usamos o IP do
// host que serve o Metro bundler. Pode ser sobrescrito por
// EXPO_PUBLIC_API_URL (ex.: build apontando para produção).
function resolveBaseUrl(): string {
  const fromEnv = process.env.EXPO_PUBLIC_API_URL;
  if (fromEnv) return fromEnv.replace(/\/$/, '');

  const hostUri = Constants.expoConfig?.hostUri ?? Constants.expoGoConfig?.debuggerHost;
  const host = hostUri?.split(':')[0];
  if (host) return `http://${host}:8082`;

  return 'http://localhost:8082';
}

export const API_BASE_URL = resolveBaseUrl();

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

interface AuthResponse {
  access_token: string;
  refresh_token: string;
  user: SessionUser;
}

async function rawRequest(path: string, init: RequestInit): Promise<Response> {
  return fetch(`${API_BASE_URL}/api/v1${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(init.headers ?? {}),
    },
  });
}

async function parseError(res: Response, fallback: string): Promise<string> {
  try {
    const body = await res.json();
    return body?.error ?? fallback;
  } catch {
    return fallback;
  }
}

// Troca o refresh token por um access token novo. Retorna false quando a
// sessão expirou de vez (o chamador deve mandar o usuário para o login).
async function refreshSession(): Promise<boolean> {
  const refreshToken = await getRefreshToken();
  if (!refreshToken) return false;

  const res = await rawRequest('/auth/refresh', {
    method: 'POST',
    body: JSON.stringify({ refresh_token: refreshToken }),
  });
  if (!res.ok) {
    await clearSession();
    return false;
  }

  const data: AuthResponse = await res.json();
  setAccessToken(data.access_token);
  await saveSession(data.refresh_token, data.user);
  return true;
}

// Requisição autenticada com um único retry após refresh — mesmo contrato do
// interceptor do cliente web.
export async function apiRequest<T>(path: string, init: RequestInit = {}): Promise<T> {
  const withAuth = (): RequestInit => {
    const token = getAccessToken();
    return {
      ...init,
      headers: {
        ...(init.headers ?? {}),
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
    };
  };

  let res = await rawRequest(path, withAuth());

  if (res.status === 401) {
    const refreshed = await refreshSession();
    if (!refreshed) {
      throw new ApiError(401, 'Sessão expirada');
    }
    res = await rawRequest(path, withAuth());
  }

  if (!res.ok) {
    throw new ApiError(res.status, await parseError(res, 'Falha na requisição'));
  }

  if (res.status === 204) {
    return undefined as T;
  }
  return res.json() as Promise<T>;
}

// ─── Auth ─────────────────────────────────────────────────────────────────────

export async function login(email: string, password: string): Promise<SessionUser> {
  const res = await rawRequest('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
  if (!res.ok) {
    throw new ApiError(res.status, await parseError(res, 'E-mail ou senha inválidos'));
  }

  const data: AuthResponse = await res.json();
  setAccessToken(data.access_token);
  await saveSession(data.refresh_token, data.user);
  return data.user;
}

export async function logout(): Promise<void> {
  const refreshToken = await getRefreshToken();
  if (refreshToken) {
    try {
      await rawRequest('/auth/logout', {
        method: 'POST',
        body: JSON.stringify({ refresh_token: refreshToken }),
      });
    } catch {
      // Logout local acontece de qualquer forma
    }
  }
  await clearSession();
}

// Restaura a sessão na abertura do app (o access token não é persistido).
export async function restoreSession(): Promise<boolean> {
  return refreshSession();
}
