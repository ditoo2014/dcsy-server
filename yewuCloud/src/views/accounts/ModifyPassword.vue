<template>
  <div class="modifyPasswordCss">
    <div class="formContentCss">
      <el-alert v-if="forcePasswordChange" :title="forcePasswordMessage" type="warning" :closable="false" show-icon />
      <el-form ref="dataForm" :rules="rules" :model="form" size="small" label-width="120px" class="password-policy-form">
        <el-row>
          <el-col :span="20">
            <el-form-item label="账号">
              <el-input v-model="form.username" disabled />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="20">
            <el-form-item label="旧密码" prop="password">
              <el-input v-model="form.password" type="password" show-password autocomplete="current-password" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="20">
            <el-form-item label="新密码" prop="newPassword">
              <el-input v-model="form.newPassword" type="password" show-password autocomplete="new-password" @input="clearPasswordPolicyServerError('newPassword')" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="20">
            <el-form-item label="确认新密码" prop="sureNewPassword">
              <el-input v-model="form.sureNewPassword" type="password" show-password autocomplete="new-password" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <div class="buttonContentCss">
        <el-button :icon="$legacyIcon('el-icon-back')" @click="backPage">
          {{ forcePasswordChange ? '退出登录' : '返回' }}
        </el-button>
        <el-button @click="clear">
          清空
        </el-button>
        <el-button type="primary" :loading="submitting" @click="setData">
          确定
        </el-button>
      </div>
    </div>
  </div>

</template>

<script>
import { goBack, getRSApassword } from '@/utils/utils.js'
import { getToken, getUsername } from '@/utils/auth.js'
import { modifyPassword } from '@/api/user.js'
import { expireSession } from '@/utils/sessionExpiration'
import { passwordChangeMessage } from '@/utils/passwordChangeState.js'
import {
  createPasswordPolicyValidator,
  PASSWORD_POLICY_ERROR_CODE,
  PASSWORD_POLICY_MESSAGE
} from '@/utils/passwordPolicy.js'

export default {
  data() {
    const validatePolicy = createPasswordPolicyValidator(
      () => this.form,
      value => this.getPasswordPolicyServerMessage('newPassword', value)
    )
    const validateNewPassword = (rule, value, callback) => {
      if (value && value === this.form.password) {
        callback(new Error('新密码不能与旧密码相同'))
        return
      }
      validatePolicy(rule, value, callback)
    }
    const validateSureNewPassword = (rule, value, callback) => { // 重复新密码
      if (this.form.newPassword !== value) {
        callback(new Error('两次输入的新密码不同'))
      } else {
        callback()
      }
    }
    return {
      submitting: false,
      passwordPolicyServerValidation: {
        fieldName: '',
        value: '',
        message: ''
      },
      rules: {
        username: [{ required: true, trigger: 'blur', message: '用户名不能为空' }],
        password: [{ required: true, trigger: 'blur', message: '旧密码不能为空' }],
        newPassword: [{ required: true, trigger: 'blur', validator: validateNewPassword }],
        sureNewPassword: [{ required: true, trigger: 'blur', validator: validateSureNewPassword }]
      },
      form: {
        username: getUsername(),
        password: '',
        newPassword: '',
        sureNewPassword: ''
      }
    }
  },
  computed: {
    forcePasswordChange() {
      return this.$store.state.user.passwordChangeRequired
    },
    forcePasswordMessage() {
      return passwordChangeMessage(this.$store.state.user.passwordChangeReason)
    }
  },
  methods: {
    setData() {
      if (this.submitting) return
      this.$refs['dataForm'].validate(valid => {
        if (valid) {
          this.submitting = true
          const form = JSON.parse(JSON.stringify(this.form))
          const requestedToken = getToken()
          const submittedPassword = form.newPassword
          form.password = getRSApassword(form.password, this)
          form.newPassword = getRSApassword(form.newPassword, this)
          modifyPassword(form).then(async res => {
            if (getToken() !== requestedToken) return
            if (res.code === 200) {
              this.$notify({
                title: '密码修改成功',
                message: '请使用新密码重新登录',
                type: 'success',
                duration: 3 * 1000
              })
              this.clear()
              // 后端已使旧令牌失效，同步清理业务、采集及其他标签页的会话。
              await expireSession({ redirectPath: '/' })
            }
          }).catch(error => {
            this.handlePasswordPolicyError(error, 'newPassword', submittedPassword)
          }).finally(() => {
            this.submitting = false
          })
        }
      })
    },
    clear() {
      this.clearPasswordPolicyServerError()
      this.form = {
        username: getUsername(),
        password: '',
        newPassword: '',
        sureNewPassword: ''
      }
    },
    getPasswordPolicyServerMessage(fieldName, value) {
      const validation = this.passwordPolicyServerValidation
      if (validation.fieldName === fieldName && validation.value === value) {
        return validation.message
      }
      return ''
    },
    clearPasswordPolicyServerError(fieldName) {
      const validation = this.passwordPolicyServerValidation
      if (!validation.message || (fieldName && validation.fieldName !== fieldName)) {
        return
      }
      const validatedField = validation.fieldName
      this.passwordPolicyServerValidation = {
        fieldName: '',
        value: '',
        message: ''
      }
      if (this.$refs.dataForm && validatedField) {
        this.$refs.dataForm.clearValidate(validatedField)
      }
    },
    handlePasswordPolicyError(error, fieldName, rejectedValue) {
      if (error && error.errorCode === PASSWORD_POLICY_ERROR_CODE) {
        this.passwordPolicyServerValidation = {
          fieldName,
          value: rejectedValue,
          message: PASSWORD_POLICY_MESSAGE
        }
        this.$nextTick(() => {
          if (this.$refs.dataForm) {
            this.$refs.dataForm.validateField(fieldName, () => {})
          }
        })
      }
    },
    async backPage() {
      if (this.forcePasswordChange) {
        await expireSession({ notifyBackend: true, redirectPath: '/' })
        return
      }
      goBack(this)
    }
  }
}
</script>

<style lang="stylus" scoped>
.modifyPasswordCss {
  width 100%
  height 100%
  // 独立改密页面在高缩放或矮窗口下仍可滚动到提交按钮。
  overflow auto
  .formContentCss {
    margin 0 auto
    padding-top 10%
    width 500px
    max-width calc(100% - 24px)
    min-height 400px
    text-align center
    .buttonContentCss {
      margin-top 30px
    }
    .password-policy-form {
      :deep(.el-form-item__content) {
        align-items flex-start
      }
      :deep(.el-form-item__error) {
        position static
        width 100%
        padding-top 6px
        color #F56C6C
        line-height 18px
        text-align left
        white-space normal
      }
    }
  }
}
</style>
