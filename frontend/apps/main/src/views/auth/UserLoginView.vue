<template>
  <AuthLayout>
    <Card
      class="bg-card box"
      :class="{ 'animate-shake': shakeCard }"
      id="login-container"
      ref="cardRef"
    >
      <CardContent class="p-6 space-y-5">
        <div class="space-y-1 text-center">
          <CardTitle class="text-2xl font-bold text-foreground">
            {{ appSettingsStore.public_config?.['app.site_name'] || 'libredesk' }}
          </CardTitle>
          <p class="text-sm text-muted-foreground">{{ t('auth.signIn') }}</p>
        </div>

        <div v-if="clerkPublishableKey" class="space-y-3">
          <div ref="clerkSignInRef" class="flex justify-center"></div>
          <p v-if="clerkLoading" class="text-sm text-center text-muted-foreground">
            {{ t('auth.loggingIn') }}
          </p>
          <button
            v-if="localLoginEnabled && !showPasswordForm"
            type="button"
            class="w-full text-sm text-muted-foreground hover:text-foreground transition-colors"
            @click="showPasswordForm = true"
          >
            {{ t('auth.signInWithPassword') }}
          </button>
        </div>

        <div v-else-if="enabledOIDCProviders.length" class="space-y-3">
          <Button
            v-for="oidcProvider in enabledOIDCProviders"
            :key="oidcProvider.id"
            variant="outline"
            type="button"
            @click="redirectToOIDC(oidcProvider)"
            class="w-full"
          >
            <img
              :src="oidcProvider.logo_url"
              :alt="oidcProvider.name"
              width="20"
              v-if="oidcProvider.logo_url"
            />
            {{ oidcProvider.name }}
          </Button>

          <div v-if="localLoginEnabled" class="relative">
            <div class="absolute inset-0 flex items-center">
              <span class="w-full border-t border-border"></span>
            </div>
            <div class="relative flex justify-center text-xs uppercase">
              <span class="px-2 text-muted-foreground bg-card">{{ t('auth.orContinueWith') }}</span>
            </div>
          </div>
        </div>

        <form
          v-if="localLoginEnabled && (!clerkPublishableKey || showPasswordForm)"
          @submit.prevent="loginAction"
          class="space-y-3"
        >
          <div class="space-y-2">
            <Label for="email" class="text-muted-foreground">{{ t('globals.terms.email') }}</Label>
            <Input
              id="email"
              type="text"
              autofocus
              autocomplete="username"
              v-model.trim="loginForm.email"
              :class="{ 'border-destructive': emailHasError }"
            />
          </div>

          <div class="space-y-2">
            <Label for="password" class="text-muted-foreground">
              {{ t('globals.terms.password') }}
            </Label>
            <div class="relative">
              <Input
                id="password"
                :type="showPassword ? 'text' : 'password'"
                autocomplete="current-password"
                v-model="loginForm.password"
                :class="{ 'border-destructive': passwordHasError }"
                class="pr-10"
              />
              <button
                type="button"
                :aria-label="showPassword ? t('auth.hidePassword') : t('auth.showPassword')"
                class="absolute inset-y-0 right-0 flex items-center pr-3 text-muted-foreground hover:text-foreground"
                @click="showPassword = !showPassword"
              >
                <Eye v-if="!showPassword" class="w-5 h-5" />
                <EyeOff v-else class="w-5 h-5" />
              </button>
            </div>
          </div>

          <div class="flex items-center justify-between">
            <router-link
              to="/reset-password"
              class="text-sm text-muted-foreground hover:text-foreground transition-colors"
            >
              {{ t('auth.forgotPassword') }}
            </router-link>
          </div>

          <Button class="w-full" :disabled="isLoading" type="submit">
            <span v-if="isLoading" class="flex items-center justify-center">
              <div
                class="w-5 h-5 border-2 border-primary-foreground/30 border-t-primary-foreground rounded-full animate-spin mr-3"
              ></div>
              {{ t('auth.loggingIn') }}
            </span>
            <span v-else>{{ t('auth.signInButton') }}</span>
          </Button>
        </form>

        <Error
          v-if="errorMessage"
          :errorMessage="errorMessage"
          :border="true"
          class="w-full bg-destructive/10 text-destructive border-destructive/20 p-3 rounded-md text-sm"
        />
      </CardContent>
    </Card>
  </AuthLayout>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, computed, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '../../api'
import { validateEmail } from '@shared-ui/utils/string'
import { applyTemporaryClass } from '@/utils/temporary-class'
import { Button } from '@shared-ui/components/ui/button'
import { Error } from '@shared-ui/components/ui/error'
import { Card, CardContent, CardTitle } from '@shared-ui/components/ui/card'
import { Input } from '@shared-ui/components/ui/input'
import { Label } from '@shared-ui/components/ui/label'
import { useEmitter } from '../../composables/useEmitter'
import { useUserStore } from '../../stores/user'
import { useI18n } from 'vue-i18n'
import { EMITTER_EVENTS } from '../../constants/emitterEvents.js'
import { useAppSettingsStore } from '../../stores/appSettings'
import AuthLayout from '@/layouts/auth/AuthLayout.vue'
import { Eye, EyeOff } from 'lucide-vue-next'
import {
  clearSignedOut,
  isMarkedSignedOut,
  keepLoginError,
  loadClerk,
  takeLoginError
} from '@main/utils/clerk.js'

const emitter = useEmitter()
const { t } = useI18n()
const errorMessage = ref('')
const isLoading = ref(false)
const router = useRouter()
const userStore = useUserStore()
const shakeCard = ref(false)
const submitted = ref(false)
const showPassword = ref(false)
const loginForm = ref({
  email: '',
  password: ''
})
const oidcProviders = ref([])
const appSettingsStore = useAppSettingsStore()
const clerkSignInRef = ref(null)
const clerkLoading = ref(false)
const showPasswordForm = ref(false)
let clerk = null
let removeClerkListener = null
// clerkActive turns false when the view unmounts or a password login starts,
// so a late Clerk session never replaces that login.
let clerkActive = true
let unmounted = false
let signInMounted = false

// Demo build has the credentials prefilled.
const isDemoBuild = import.meta.env.VITE_DEMO_BUILD === 'true'

const demoCredentials = {
  email: 'demo@libredesk.io',
  password: 'demo@libredesk.io'
}

const oidcErrorKeys = {
  oidc_invalid_client: 'auth.oidcInvalidClient',
  oidc_access_denied: 'auth.oidcAccessDenied',
  oidc_session_expired: 'auth.oidcSessionExpired',
  oidc_no_account: 'auth.oidcNoAccount',
  oidc_account_disabled: 'user.accountDisabled',
  oidc_login_failed: 'auth.oidcLoginFailed'
}

onMounted(async () => {
  // Prefill the login form with demo credentials if it's a demo build
  if (isDemoBuild) {
    loginForm.value.email = demoCredentials.email
    loginForm.value.password = demoCredentials.password
  }
  fetchOIDCProviders()
  showOIDCError()
  if (clerkPublishableKey.value) {
    showKeptLoginError()
    initClerk()
  }
})

onBeforeUnmount(() => {
  unmounted = true
  clerkActive = false
  removeClerkListener?.()
  if (clerk && clerkSignInRef.value) clerk.unmountSignIn(clerkSignInRef.value)
})

const showKeptLoginError = () => {
  const keptError = takeLoginError(window.sessionStorage)
  if (keptError) {
    errorMessage.value = keptError
    applyTemporaryClass('login-container', 'animate-shake')
  }
}

const initClerk = async () => {
  try {
    clerk = await loadClerk(clerkPublishableKey.value)
    if (!clerkActive) return
    // Logging out of Libredesk ends the Clerk session too. The marker stays
    // until Clerk confirms the session is gone.
    if (isMarkedSignedOut(window.sessionStorage)) {
      if (clerk.session) await clerk.signOut()
      if (clerk.session) return
      clearSignedOut(window.sessionStorage)
    }
    if (!clerkActive) return
    listenForClerkSession()
    if (clerk.session) {
      await exchangeClerkSession()
    } else {
      await mountClerkSignIn()
    }
  } catch (error) {
    if (!clerkActive) return
    errorMessage.value = handleHTTPError(error).message
    showPasswordForm.value = true
  }
}

const listenForClerkSession = () => {
  removeClerkListener?.()
  removeClerkListener = clerk.addListener(({ session }) => {
    if (session) exchangeClerkSession()
  })
}

const mountClerkSignIn = async () => {
  await nextTick()
  if (!clerk || !clerkSignInRef.value || signInMounted) return
  signInMounted = true
  clerk.mountSignIn(clerkSignInRef.value, {
    forceRedirectUrl: window.location.href,
    appearance: {
      // Follow the Libredesk theme, light or dark.
      variables: {
        colorBackground: 'hsl(var(--card))',
        colorForeground: 'hsl(var(--card-foreground))',
        colorMutedForeground: 'hsl(var(--muted-foreground))',
        colorPrimary: 'hsl(var(--primary))',
        colorPrimaryForeground: 'hsl(var(--primary-foreground))',
        colorDanger: 'hsl(var(--destructive))',
        colorInput: 'hsl(var(--card))',
        colorInputForeground: 'hsl(var(--foreground))',
        colorBorder: 'hsl(var(--input))',
        colorNeutral: 'hsl(var(--foreground))',
        borderRadius: 'var(--radius)'
      },
      elements: {
        rootBox: 'w-full',
        cardBox: 'w-full shadow-none border-0',
        card: 'shadow-none px-0 py-2',
        header: 'hidden',
        footer: 'hidden',
        formFieldInput: '!border !border-solid !border-input',
        socialButtonsBlockButton: '!border !border-solid !border-input'
      }
    }
  })
}

// exchangeClerkSession trades the Clerk session token for a Libredesk session.
const exchangeClerkSession = async () => {
  if (!clerkActive || clerkLoading.value || !clerk?.session) return
  if (isMarkedSignedOut(window.sessionStorage)) return
  clerkLoading.value = true
  errorMessage.value = ''
  try {
    const token = await clerk.session.getToken()
    if (!clerkActive) return
    const resp = await api.clerkLogin({ token })
    if (!clerkActive) return
    completeLogin(resp)
  } catch (error) {
    if (!clerkActive) return
    errorMessage.value = handleHTTPError(error).message
    applyTemporaryClass('login-container', 'animate-shake')
    // End the Clerk session so the agent can try another account. Clerk can
    // reload the page when it signs out, so keep the error for the next load.
    keepLoginError(window.sessionStorage, errorMessage.value)
    await clerk.signOut().catch(() => {})
    await mountClerkSignIn()
  } finally {
    clerkLoading.value = false
  }
}

const completeLogin = (resp) => {
  takeLoginError(window.sessionStorage)
  if (resp?.data?.data) {
    userStore.setCurrentUser(resp.data.data)
  }
  // Also fetch general setting as user's logged in.
  appSettingsStore.fetchSettings('general')

  // Redirect to the 'next' parameter if it exists
  const nextParam = router.currentRoute.value.query.next
  if (nextParam) {
    router.push(nextParam)
  } else {
    router.push({ name: 'inboxes' })
  }
}

const showOIDCError = () => {
  const { error, ...query } = router.currentRoute.value.query
  if (!error) return
  errorMessage.value = t(Object.hasOwn(oidcErrorKeys, error) ? oidcErrorKeys[error] : 'auth.oidcLoginFailed')
  applyTemporaryClass('login-container', 'animate-shake')
  router.replace({ query })
}

const fetchOIDCProviders = async () => {
  try {
    const config = appSettingsStore.public_config
    if (config && config['app.sso_providers']) {
      oidcProviders.value = config['app.sso_providers'] || []
    }
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const redirectToOIDC = (provider) => {
  // Pass the 'next' parameter to OIDC login if it exists
  const nextParam = router.currentRoute.value.query.next
  const url = nextParam
    ? `/api/v1/oidc/${provider.id}/login?next=${encodeURIComponent(nextParam)}`
    : `/api/v1/oidc/${provider.id}/login`
  window.location.href = url
}

const validateForm = () => {
  if (!validateEmail(loginForm.value.email) && loginForm.value.email !== 'System') {
    errorMessage.value = t('validation.invalidEmail')
    applyTemporaryClass('login-container', 'animate-shake')
    return false
  }
  if (!loginForm.value.password) {
    errorMessage.value = t('validation.passwordCannotBeEmpty')
    applyTemporaryClass('login-container', 'animate-shake')
    return false
  }
  return true
}

const loginAction = () => {
  submitted.value = true
  if (!validateForm()) return

  errorMessage.value = ''
  isLoading.value = true
  // A password login wins over a Clerk session that loads later.
  clerkActive = false
  removeClerkListener?.()

  api
    .login({
      email: loginForm.value.email,
      password: loginForm.value.password
    })
    .then(completeLogin)
    .catch((error) => {
      errorMessage.value = handleHTTPError(error).message
      applyTemporaryClass('login-container', 'animate-shake')
      // The password login failed, so Clerk sign-in works again. Clerk may
      // still have been loading, so run its setup again.
      if (clerkPublishableKey.value && !unmounted) {
        takeLoginError(window.sessionStorage)
        clerkActive = true
        initClerk()
      }
    })
    .finally(() => {
      isLoading.value = false
    })
}

const enabledOIDCProviders = computed(() => {
  return oidcProviders.value.filter((provider) => !provider.disabled)
})

const clerkPublishableKey = computed(
  () => appSettingsStore.public_config?.['auth.clerk_publishable_key'] || ''
)

const localLoginEnabled = computed(
  () => appSettingsStore.public_config?.['auth.local_login_enabled'] !== false
)

const emailHasError = computed(() => {
  if (!submitted.value) return false
  const email = loginForm.value.email
  return email !== 'System' && !validateEmail(email)
})

const passwordHasError = computed(() => submitted.value && !loginForm.value.password)
</script>
