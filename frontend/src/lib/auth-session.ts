type TokenResponse = {
  data?: {
    access_token?: string;
    refresh_token?: string;
  };
  error?: {
    message?: string;
  };
};

export async function refreshAccessToken(): Promise<string | null> {
  const refresh = sessionStorage.getItem("asutport_refresh_token");
  if (!refresh) {
    return null;
  }
  const response = await fetch("/api/v1/auth/refresh", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: refresh }),
  });
  const body = (await response.json()) as TokenResponse;
  if (!response.ok || !body.data?.access_token) {
    return null;
  }
  sessionStorage.setItem("asutport_access_token", body.data.access_token);
  if (body.data.refresh_token) {
    sessionStorage.setItem("asutport_refresh_token", body.data.refresh_token);
  }
  return body.data.access_token;
}

export async function ensureAccessToken(): Promise<string | null> {
  const current = sessionStorage.getItem("asutport_access_token");
  if (current) {
    return current;
  }
  return refreshAccessToken();
}

export async function authFetch(input: string, init: RequestInit = {}, retry = true): Promise<Response> {
  const token = await ensureAccessToken();
  const headers = new Headers(init.headers);
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  const response = await fetch(input, { ...init, headers });
  if (response.status === 401 && retry) {
    const refreshed = await refreshAccessToken();
    if (refreshed) {
      return authFetch(input, init, false);
    }
  }
  return response;
}

export function defaultCorsHints(site = "https://asutport.ru") {
  const origins = [site, "http://localhost:3000", "http://127.0.0.1:3000"];
  const corsXml = `<CORSConfiguration>
  <CORSRule>
    <AllowedOrigin>${site}</AllowedOrigin>
    <AllowedOrigin>http://localhost:3000</AllowedOrigin>
    <AllowedOrigin>http://127.0.0.1:3000</AllowedOrigin>
    <AllowedMethod>GET</AllowedMethod>
    <AllowedMethod>PUT</AllowedMethod>
    <AllowedMethod>POST</AllowedMethod>
    <AllowedMethod>DELETE</AllowedMethod>
    <AllowedMethod>HEAD</AllowedMethod>
    <AllowedHeader>*</AllowedHeader>
    <ExposeHeader>ETag</ExposeHeader>
  </CORSRule>
</CORSConfiguration>`;
  return { allowed_origins: origins, cors_xml: corsXml };
}
