/** Pink capsule label: −10%=¥558 or −¥100=¥1,000 */

export function formatYenPlain(yen) {
  return `¥${Number(yen || 0).toLocaleString('en-US')}`
}

export function discountBadgeText(listCents, unitCents, storedLabel = '') {
  if (storedLabel && /[=＝]/.test(storedLabel) && /[¥%]/.test(storedLabel)) {
    return storedLabel
  }
  const list = Number(listCents) || 0
  const unit = Number(unitCents) || 0
  if (!(list > unit)) return ''

  for (let p = 1; p <= 100; p++) {
    if (list - Math.floor((list * p) / 100) === unit) {
      return `−${p}%=${formatYenPlain(unit)}`
    }
  }
  const off = list - unit
  return `−${formatYenPlain(off)}=${formatYenPlain(unit)}`
}

export function hasDiscount(listCents, unitCents) {
  return Number(listCents) > Number(unitCents)
}
