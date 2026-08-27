import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { resolveGuard } from './guards'
import FrontLayout from '@/components/layout/FrontLayout.vue'
import AdminLayout from '@/components/layout/AdminLayout.vue'
import HomePage from '@/pages/home/index.vue'
import LoginPage from '@/pages/auth/LoginPage.vue'
import RegisterPage from '@/pages/auth/RegisterPage.vue'
import ResourceDetailPage from '@/pages/resource/ResourceDetailPage.vue'
import SearchPage from '@/pages/search/SearchPage.vue'
import ProfilePage from '@/pages/user/ProfilePage.vue'
import AdminDashboard from '@/pages/admin/AdminDashboard.vue'
import AdminUsers from '@/pages/admin/AdminUsers.vue'
import PlaceholderPage from '@/pages/placeholder/PlaceholderPage.vue'
import NotFoundPage from '@/pages/not-found/NotFound.vue'

const routes = [
  {
    path: '/',
    component: FrontLayout,
    children: [
      { path: '', name: 'home', component: HomePage, meta: { requiresAuth: true } },
      { path: 'search', name: 'search', component: SearchPage, meta: { requiresAuth: true } },
      { path: 'resources/:id', name: 'resource-detail', component: ResourceDetailPage, meta: { requiresAuth: true } },
      { path: 'user/me', name: 'user-me', component: ProfilePage, meta: { requiresAuth: true } },
      {
        path: 'resources/upload',
        name: 'resource-upload',
        component: PlaceholderPage,
        meta: { requiresAuth: true, requiresAdmin: true },
      },
    ],
  },
  {
    path: '/login',
    name: 'login',
    component: LoginPage,
    meta: { guestOnly: true },
  },
  {
    path: '/register',
    name: 'register',
    component: RegisterPage,
    meta: { guestOnly: true },
  },
  {
    path: '/admin',
    component: AdminLayout,
    children: [
      {
        path: '',
        name: 'admin-dashboard',
        component: AdminDashboard,
        meta: { requiresAuth: true, requiresAdmin: true },
      },
      {
        path: 'users',
        name: 'admin-users',
        component: AdminUsers,
        meta: { requiresAuth: true, requiresAdmin: true },
      },
      {
        path: 'resources',
        name: 'admin-resources',
        component: PlaceholderPage,
        meta: { requiresAuth: true, requiresAdmin: true },
      },
      {
        path: 'categories',
        name: 'admin-categories',
        component: PlaceholderPage,
        meta: { requiresAuth: true, requiresAdmin: true },
      },
    ],
  },
  { path: '/:pathMatch(.*)*', name: 'not-found', component: NotFoundPage },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  return resolveGuard(to, auth)
})

export default router
