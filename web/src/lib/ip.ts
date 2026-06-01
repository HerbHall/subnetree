// IP address helpers — pure functions, no side effects.
//
// Why this exists: the devices table sorted IP addresses as plain strings,
// so 192.168.1.111 sorted before 192.168.1.12 (lexicographically "111" < "12").
// This module provides a numeric, octet-aware comparator instead. See issue #585.

/**
 * Parse a dotted-quad IPv4 string into its four numeric octets.
 * Returns null for anything that is not a valid IPv4 address
 * (empty string, IPv6, hostnames, out-of-range octets, etc.).
 */
export function parseIpv4(ip: string): [number, number, number, number] | null {
  const parts = ip.split('.')
  if (parts.length !== 4) return null

  const octets: number[] = []
  for (const part of parts) {
    // Reject empty, non-numeric, or values with leading/trailing junk.
    if (part === '' || !/^\d{1,3}$/.test(part)) return null
    const n = Number(part)
    if (n < 0 || n > 255) return null
    octets.push(n)
  }
  return octets as [number, number, number, number]
}

/**
 * Compare two IP address strings for sorting in ascending order.
 *
 * Returns a negative number if `a` should come before `b`, a positive
 * number if `a` should come after `b`, and 0 if they are equal — the
 * standard Array.prototype.sort() comparator contract.
 *
 * Design decisions this function encodes:
 *   1. Two valid IPv4 addresses compare numerically, octet by octet,
 *      so 192.168.1.12 sorts before 192.168.1.111.
 *   2. Values that are NOT valid IPv4 (empty, IPv6, malformed) fall back
 *      to a deterministic, locale-independent string comparison so the sort
 *      stays stable and total — and they sort AFTER all valid IPv4 addresses
 *      (invalid/missing data sinks to the bottom of an ascending sort, where
 *      it's least noisy).
 *
 * Note: this comparator is direction-agnostic — it always describes
 * ascending order. The caller flips the sign for descending sorts.
 */
export function compareIpAddresses(a: string, b: string): number {
  const pa = parseIpv4(a)
  const pb = parseIpv4(b)

  if (pa && pb) {
    for (let i = 0; i < 4; i++) {
      if (pa[i] !== pb[i]) return pa[i] - pb[i]
    }
    return 0
  }

  // Exactly one is a valid IPv4 address: it sorts before the invalid one.
  if (pa) return -1
  if (pb) return 1

  // Neither is valid IPv4 (empty, IPv6, malformed): fall back to a
  // deterministic, locale-independent code-unit comparison. (localeCompare
  // would vary the order by the host's runtime locale, breaking the
  // determinism this fallback promises.) Matches the plain string sort used
  // by the devices table's other columns.
  if (a < b) return -1
  if (a > b) return 1
  return 0
}
