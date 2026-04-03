import * as authApi from "@/data/authApi"

export const authRepository = {
  authenticate: authApi.authenticate,
  checkAuthentication: authApi.checkAuthentication,
  logout: authApi.logout,
}
