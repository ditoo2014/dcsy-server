/** Vue 2 filter compatibility via globalProperties and template helpers. */
export function registerGlobalFilters(app, filters) {
  app.config.globalProperties.$filters = filters
  Object.keys(filters).forEach(key => {
    app.config.globalProperties[`$${key}`] = filters[key]
  })
}
