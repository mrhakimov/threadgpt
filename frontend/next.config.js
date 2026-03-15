/** @type {import('next').NextConfig} */
const nextConfig = {
  async headers() {
    const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8000"
    const isDev = process.env.NODE_ENV === "development"

    const csp = isDev
      ? `default-src 'self'; script-src 'self' 'unsafe-eval' 'unsafe-inline'; connect-src 'self' ${apiUrl} ws://localhost:3000; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'`
      : `default-src 'self'; script-src 'self'; connect-src 'self' ${apiUrl}; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'`

    return [
      {
        source: "/(.*)",
        headers: [
          {
            key: "Content-Security-Policy",
            value: csp,
          },
        ],
      },
    ]
  },
}

module.exports = nextConfig
