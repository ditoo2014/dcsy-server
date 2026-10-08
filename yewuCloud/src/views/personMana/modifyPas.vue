<template>
  <div class="modfiyPasCss">
    <div class="formContentCss">
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
          返回
        </el-button>
        <el-button @click="clear">
          清空
        </el-button>
        <el-button type="primary" @click="setData">
          确定
        </el-button>
      </div>
    </div>
  </div>

</template>

<script>
import { goBack, getRSApassword } from '@/utils/utils.js'
import { getUsername } from '@/utils/auth.js'
import { modifyPassword } from '@/api/user.js'
import {
  createPasswordPolicyValidator,
  PASSWORD_POLICY_ERROR_CODE,
  PASSWORD_POLICY_MESSAGE
} from '@/utils/password-policy.js'

export default {
  data() {
    const validateNewPassword = createPasswordPolicyValidator(
      () => this.form,
      value => this.getPasswordPolicyServerMessage('newPassword', value)
    )
    const validateSureNewPassword = (rule, value, callback) => { // 重复新密码
      if (this.form.newPassword !== value) {
        callback(new Error('两次输入的新密码不同'))
      } else {
        callback()
      }
    }
    return {
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
  methods: {
    setData() {
      this.$refs['dataForm'].validate(valid => {
        if (valid) {
          const form = JSON.parse(JSON.stringify(this.form))
          const submittedPassword = form.newPassword
          form.password = getRSApassword(form.password, this)
          form.newPassword = getRSApassword(form.newPassword, this)
          modifyPassword(form).then(res => {
            if (res.code === 200) {
              this.$notify({
                title: '修改请求发送成功',
                message: `成功修改`,
                type: 'success',
                duration: 3 * 1000
              })
              this.clear()
            }
          }).catch(error => {
            this.handlePasswordPolicyError(error, 'newPassword', submittedPassword)
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
    backPage() {
      goBack(this)
    }
  }
}
</script>

<style lang="stylus" scoped>
.modfiyPasCss {
  width 100%
  height 100%
  .formContentCss {
    margin 0 auto
    padding-top 10%
    width 500px
    height 400px
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
