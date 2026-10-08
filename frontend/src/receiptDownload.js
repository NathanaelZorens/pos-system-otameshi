import { jsPDF } from 'jspdf'

/** Helvetica has no yen glyph — keep PDF ASCII-safe. */
function pdfMoney(yen) {
  const n = Number(yen) || 0
  return `JPY ${n.toLocaleString('en-US')}`
}

/** Narrow slip PDF — auto-downloads, no print dialog. */
export function downloadOrderReceipt(order) {
  if (!order) return

  const isTakeout = order.source === 'takeout'
  const stamp = (order.pickup_code || order.id || 'order').toString().slice(0, 8)
  const pageW = 80
  const margin = 8
  const contentW = pageW - margin * 2
  let y = 10

  // Tall canvas; PDF viewers crop empty space fine for a slip.
  const doc = new jsPDF({ unit: 'mm', format: [pageW, 280] })
  const cx = pageW / 2

  const line = (text, opts = {}) => {
    const size = opts.size || 9
    doc.setFont('helvetica', opts.bold ? 'bold' : 'normal')
    doc.setFontSize(size)
    const lines = doc.splitTextToSize(String(text ?? ''), contentW)
    doc.text(lines, opts.center ? cx : margin, y, {
      align: opts.center ? 'center' : 'left',
    })
    y += lines.length * (size * 0.45) + (opts.gap ?? 1.5)
  }

  const rule = () => {
    y += 1
    doc.setDrawColor(180)
    doc.line(margin, y, pageW - margin, y)
    y += 4
  }

  line('OTAMESHI', { size: 11, bold: true, center: true, gap: 2 })
  line(isTakeout ? 'Takeout receipt' : 'Receipt', { size: 10, bold: true, center: true, gap: 3 })

  if (isTakeout) {
    line('Takeout', { center: true, gap: 1 })
    if (order.pickup_code) {
      line(`Pickup: ${order.pickup_code}`, { size: 14, bold: true, center: true, gap: 2 })
    }
  } else {
    line(`Table ${order.table_label || '—'}`, { size: 11, bold: true, center: true, gap: 2 })
  }

  if (order.created_at) line(order.created_at, { size: 8, center: true, gap: 1 })
  if (order.customer_name) line(`Customer: ${order.customer_name}`, { size: 8, center: true, gap: 1 })
  if (order.waiter_name && !isTakeout) line(`Staff: ${order.waiter_name}`, { size: 8, center: true, gap: 1 })

  rule()

  for (const it of order.items || []) {
    const left = `${it.quantity}x ${it.name_snapshot}`
    const right = pdfMoney(it.line_total_cents)
    doc.setFont('helvetica', 'normal')
    doc.setFontSize(9)
    const leftLines = doc.splitTextToSize(left, contentW - 22)
    doc.text(leftLines, margin, y)
    doc.text(right, pageW - margin, y, { align: 'right' })
    y += Math.max(leftLines.length * 4, 4) + 1.2
  }

  rule()

  doc.setFont('helvetica', 'bold')
  doc.setFontSize(11)
  doc.text('Total', margin, y)
  doc.text(pdfMoney(order.total_cents), pageW - margin, y, { align: 'right' })
  y += 6

  if (order.payment) {
    const pay = `Paid via ${order.payment.method}${
      order.payment.provider ? ` (${order.payment.provider})` : ''
    }`
    line(pay, { size: 8, center: true, gap: 2 })
  }

  line('Thank you.', { size: 9, center: true, gap: 2 })

  doc.save(`receipt-${stamp}.pdf`)
}
