import { describe, it, expect } from 'vitest'
import { parseIpv4, compareIpAddresses } from './ip'

describe('parseIpv4', () => {
  it('parses a valid dotted-quad into four octets', () => {
    expect(parseIpv4('192.168.1.12')).toEqual([192, 168, 1, 12])
    expect(parseIpv4('0.0.0.0')).toEqual([0, 0, 0, 0])
    expect(parseIpv4('255.255.255.255')).toEqual([255, 255, 255, 255])
  })

  it('rejects out-of-range octets', () => {
    expect(parseIpv4('192.168.1.256')).toBeNull()
    expect(parseIpv4('999.1.1.1')).toBeNull()
  })

  it('rejects malformed values', () => {
    expect(parseIpv4('')).toBeNull()
    expect(parseIpv4('192.168.1')).toBeNull()
    expect(parseIpv4('192.168.1.1.1')).toBeNull()
    expect(parseIpv4('192.168.1.')).toBeNull()
    expect(parseIpv4('192.168.1.x')).toBeNull()
    expect(parseIpv4(' 192.168.1.1')).toBeNull()
  })

  it('rejects IPv6 addresses', () => {
    expect(parseIpv4('fe80::1')).toBeNull()
    expect(parseIpv4('2001:db8::1')).toBeNull()
  })
})

describe('compareIpAddresses', () => {
  it('orders by numeric octet value, not lexicographically (issue #585)', () => {
    // The original bug: "111" < "12" as strings, so .111 sorted before .12.
    expect(compareIpAddresses('192.168.1.12', '192.168.1.111')).toBeLessThan(0)
    expect(compareIpAddresses('192.168.1.111', '192.168.1.12')).toBeGreaterThan(0)
  })

  it('sorts a realistic list of same-subnet IPs ascending', () => {
    const sorted = ['192.168.1.111', '192.168.1.2', '192.168.1.12', '192.168.1.20'].sort(
      compareIpAddresses
    )
    expect(sorted).toEqual(['192.168.1.2', '192.168.1.12', '192.168.1.20', '192.168.1.111'])
  })

  it('compares across all octets, highest-order first', () => {
    expect(compareIpAddresses('10.0.0.1', '9.255.255.255')).toBeGreaterThan(0)
    expect(compareIpAddresses('192.168.2.1', '192.168.10.1')).toBeLessThan(0)
  })

  it('returns 0 for identical addresses', () => {
    expect(compareIpAddresses('192.168.1.1', '192.168.1.1')).toBe(0)
  })

  it('sorts valid IPv4 before invalid/missing values (valid-first)', () => {
    expect(compareIpAddresses('192.168.1.1', '')).toBeLessThan(0)
    expect(compareIpAddresses('', '192.168.1.1')).toBeGreaterThan(0)
    expect(compareIpAddresses('192.168.1.1', 'fe80::1')).toBeLessThan(0)
  })

  it('falls back to string comparison when neither value is valid IPv4', () => {
    expect(compareIpAddresses('aaa', 'bbb')).toBeLessThan(0)
    expect(compareIpAddresses('fe80::2', 'fe80::1')).toBeGreaterThan(0)
    expect(compareIpAddresses('', '')).toBe(0)
  })

  it('pushes empty/invalid IPs to the bottom of an ascending sort', () => {
    const sorted = ['', '192.168.1.50', 'fe80::1', '192.168.1.5'].sort(compareIpAddresses)
    expect(sorted.slice(0, 2)).toEqual(['192.168.1.5', '192.168.1.50'])
    // The two invalid entries land last (order among them is string-stable).
    expect(sorted.slice(2).sort()).toEqual(['', 'fe80::1'].sort())
  })
})
