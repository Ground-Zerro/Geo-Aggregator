package main

import (
	"fmt"
	"io"
	"net"
	"strings"
)

func pbReadVarint(data []byte, pos int) (uint64, int, error) {
	var result uint64
	shift := 0
	for pos < len(data) {
		b := data[pos]
		pos++
		result |= uint64(b&0x7F) << shift
		if b < 0x80 {
			return result, pos, nil
		}
		shift += 7
	}
	return 0, pos, io.ErrUnexpectedEOF
}

func pbIterFields(data []byte, fn func(fieldNum, wireType int, raw []byte, varint uint64)) {
	pos := 0
	for pos < len(data) {
		tag, p, err := pbReadVarint(data, pos)
		if err != nil {
			return
		}
		pos = p
		fieldNum := int(tag >> 3)
		wireType := int(tag & 7)
		switch wireType {
		case 0:
			v, p2, err := pbReadVarint(data, pos)
			if err != nil {
				return
			}
			pos = p2
			fn(fieldNum, wireType, nil, v)
		case 2:
			length, p2, err := pbReadVarint(data, pos)
			if err != nil {
				return
			}
			end := p2 + int(length)
			if end > len(data) {
				return
			}
			fn(fieldNum, wireType, data[p2:end], 0)
			pos = end
		default:
			return
		}
	}
}

func pbAppendVarint(b []byte, v uint64) []byte {
	for v >= 0x80 {
		b = append(b, byte(v)|0x80)
		v >>= 7
	}
	return append(b, byte(v))
}

func pbAppendBytes(b []byte, fieldNum int, data []byte) []byte {
	b = pbAppendVarint(b, uint64(fieldNum<<3|2))
	b = pbAppendVarint(b, uint64(len(data)))
	return append(b, data...)
}

func pbAppendString(b []byte, fieldNum int, s string) []byte {
	return pbAppendBytes(b, fieldNum, []byte(s))
}

func pbAppendVarintField(b []byte, fieldNum int, v uint64) []byte {
	b = pbAppendVarint(b, uint64(fieldNum<<3))
	return pbAppendVarint(b, v)
}

func parseGeoSiteDat(data []byte, out tagSet) {
	pbIterFields(data, func(fn, wt int, raw []byte, _ uint64) {
		if fn != 1 || wt != 2 {
			return
		}
		var code string
		pbIterFields(raw, func(sfn, swt int, sb []byte, _ uint64) {
			if sfn == 1 && swt == 2 {
				code = strings.ToLower(string(sb))
			}
		})
		if code == "" || isCNTag(code) {
			return
		}
		pbIterFields(raw, func(sfn, swt int, sb []byte, _ uint64) {
			if sfn != 2 || swt != 2 {
				return
			}
			var domType uint64 = 2
			var domValue string
			pbIterFields(sb, func(dfn, _ int, db []byte, dv uint64) {
				if dfn == 1 {
					domType = dv
				} else if dfn == 2 {
					domValue = string(db)
				}
			})
			if domValue == "" {
				return
			}
			switch domType {
			case 2, 3:
				if strings.Contains(domValue, ".") && !cnDomainRE.MatchString(domValue) {
					out.add(code, domValue)
				}
			case 1:
				if d := extractDomainFromRegex(domValue); d != "" && !cnDomainRE.MatchString(d) {
					out.add(code, d)
				}
			}
		})
	})
}

func parseGeoIPDat(data []byte, out tagSet) {
	pbIterFields(data, func(fn, wt int, raw []byte, _ uint64) {
		if fn != 1 || wt != 2 {
			return
		}
		var code string
		pbIterFields(raw, func(sfn, swt int, sb []byte, _ uint64) {
			if sfn == 1 && swt == 2 {
				code = strings.ToLower(string(sb))
			}
		})
		if code == "" || isCNTag(code) {
			return
		}
		pbIterFields(raw, func(sfn, swt int, sb []byte, _ uint64) {
			if sfn != 2 || swt != 2 {
				return
			}
			var ipBytes []byte
			var prefix uint64
			pbIterFields(sb, func(dfn, dwt int, db []byte, dv uint64) {
				if dfn == 1 && dwt == 2 {
					ipBytes = db
				} else if dfn == 2 {
					prefix = dv
				}
			})
			if len(ipBytes) == 4 {
				out.add(code, fmt.Sprintf("%s/%d", net.IP(ipBytes).String(), prefix))
			} else if len(ipBytes) == 16 {
				out.add(code, fmt.Sprintf("%s/%d", net.IP(ipBytes).String(), prefix))
			}
		})
	})
}

func buildGeoSiteDat(tags tagSet) []byte {
	var result []byte
	for _, tag := range tags.sortedKeys() {
		entries := tags[tag].sorted()
		var domains []string
		for _, e := range entries {
			if !isIPEntry(e) {
				domains = append(domains, e)
			}
		}
		if len(domains) == 0 {
			continue
		}
		var site []byte
		site = pbAppendString(site, 1, strings.ToUpper(tag))
		for _, d := range domains {
			var dom []byte
			dom = pbAppendVarintField(dom, 1, 2)
			dom = pbAppendString(dom, 2, d)
			site = pbAppendBytes(site, 2, dom)
		}
		result = pbAppendBytes(result, 1, site)
	}
	return result
}

func buildGeoIPDat(tags tagSet) []byte {
	var result []byte
	for _, tag := range tags.sortedKeys() {
		entries := tags[tag].sorted()
		var cidrs []string
		for _, e := range entries {
			if isIPEntry(e) {
				cidrs = append(cidrs, e)
			}
		}
		if len(cidrs) == 0 {
			continue
		}
		var geoip []byte
		geoip = pbAppendString(geoip, 1, strings.ToUpper(tag))
		for _, cidr := range cidrs {
			cidrBytes, err := encodeCIDR(cidr)
			if err != nil {
				continue
			}
			geoip = pbAppendBytes(geoip, 2, cidrBytes)
		}
		result = pbAppendBytes(result, 1, geoip)
	}
	return result
}

func encodeCIDR(cidrStr string) ([]byte, error) {
	_, ipNet, err := net.ParseCIDR(cidrStr)
	if err != nil {
		ip := net.ParseIP(cidrStr)
		if ip == nil {
			return nil, fmt.Errorf("invalid IP/CIDR: %s", cidrStr)
		}
		var b []byte
		if ip4 := ip.To4(); ip4 != nil {
			b = pbAppendBytes(b, 1, ip4)
			b = pbAppendVarintField(b, 2, 32)
		} else {
			b = pbAppendBytes(b, 1, ip.To16())
			b = pbAppendVarintField(b, 2, 128)
		}
		return b, nil
	}
	ones, _ := ipNet.Mask.Size()
	var b []byte
	if ip4 := ipNet.IP.To4(); ip4 != nil {
		b = pbAppendBytes(b, 1, ip4)
	} else {
		b = pbAppendBytes(b, 1, ipNet.IP.To16())
	}
	b = pbAppendVarintField(b, 2, uint64(ones))
	return b, nil
}
