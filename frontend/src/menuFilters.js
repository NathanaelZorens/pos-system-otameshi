/** Build filter chips from menu items that have a category. */
export function categoryFiltersFromMenu(menu) {
  const map = new Map()
  for (const m of menu || []) {
    const id = m.category_id || m.category || ''
    const name = m.category || 'Uncategorized'
    if (!m.category_id && !m.category) continue
    if (!map.has(id)) map.set(id, name)
  }
  return [...map.entries()]
    .map(([id, name]) => ({ id, name }))
    .sort((a, b) => a.name.localeCompare(b.name))
}

export function filterMenuByCategory(menu, categoryFilter) {
  if (!categoryFilter) return menu || []
  return (menu || []).filter((m) => {
    const id = m.category_id || m.category || ''
    return id === categoryFilter
  })
}
