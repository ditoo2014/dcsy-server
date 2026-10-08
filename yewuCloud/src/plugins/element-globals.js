/** Map element-ui style globals ($message, $confirm, ...) onto element-plus. */

/** Element Plus MessageBox 点 X / 取消时会 reject('cancel'|'close')，需吞掉以免开发环境报 Uncaught runtime error */
function isMessageBoxDismiss(err) {
  return err === 'cancel' || err === 'close'
}

function wrapMessageBox(method) {
  return (...args) => {
    const promise = method(...args)
    // 仅吞掉用户关闭弹框的 reject，避免 overlay；不影响调用方自行 .catch()
    promise.catch(err => {
      if (isMessageBoxDismiss(err)) {
        return
      }
      console.error(err)
    })
    return promise
  }
}

export function registerElementGlobals(app, { ElMessage, ElMessageBox, ElNotification, ElLoading }) {
  app.config.globalProperties.$message = ElMessage
  app.config.globalProperties.$notify = ElNotification
  app.config.globalProperties.$loading = ElLoading.service
  app.config.globalProperties.$confirm = wrapMessageBox(ElMessageBox.confirm)
  app.config.globalProperties.$alert = wrapMessageBox(ElMessageBox.alert)
  app.config.globalProperties.$prompt = wrapMessageBox(ElMessageBox.prompt)
}
