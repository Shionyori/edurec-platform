import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { pageLoading } from '@/composables/pageLoading'
import { resolveGuard } from './guards'
import FrontLayout from '@/components/layout/FrontLayout.vue'
import AdminLayout from '@/components/layout/AdminLayout.vue'
import HomePage from '@/pages/home/index.vue'
import LoginPage from '@/pages/auth/LoginPage.vue'
import RegisterPage from '@/pages/auth/RegisterPage.vue'
import ResourceDetailPage from '@/pages/resource/ResourceDetailPage.vue'
import SearchPage from '@/pages/search/SearchPage.vue'
import KnowledgeDetailPage from '@/pages/knowledge/DetailPage.vue'
import DisciplineListPage from '@/pages/discipline/index.vue'
import DisciplineDetailPage from '@/pages/discipline/DetailPage.vue'
import QuestionnairePage from '@/pages/questionnaire/index.vue'
import LearningPathPage from '@/pages/learning-path/DetailPage.vue'
import ProfilePage from '@/pages/user/ProfilePage.vue'
import UploadPage from '@/pages/upload/UploadPage.vue'
import AdminDashboard from '@/pages/admin/AdminDashboard.vue'
import AdminUsers from '@/pages/admin/AdminUsers.vue'
import AdminResources from '@/pages/admin/AdminResources.vue'
import AdminCategories from '@/pages/admin/AdminCategories.vue'
import AdminRecommendations from '@/pages/admin/AdminRecommendations.vue'
import NotFoundPage from '@/pages/not-found/NotFound.vue'

const routes = [
  {
    path: '/',
    component: FrontLayout,
    children: [
      { path: '', name: 'home', component: HomePage, meta: { requiresAuth: true } },
      { path: 'search', name: 'search', component: SearchPage, meta: { requiresAuth: true } },
      { path: 'knowledge', name: 'knowledge', component: DisciplineListPage, meta: { requiresAuth: true } },
      {
        path: 'discipline/:code',
        name: 'discipline',
        component: DisciplineDetailPage,
        meta: { requiresAuth: true },
      },
      {
        path: 'knowledge/:code',
        name: 'knowledge-detail',
        component: KnowledgeDetailPage,
        meta: { requiresAuth: true },
      },
      { path: 'questionnaire', name: 'questionnaire', component: QuestionnairePage, meta: { requiresAuth: true } },
      {
        path: 'learning-path/:id',
        name: 'learning-path',
        component: LearningPathPage,
        meta: { requiresAuth: true },
      },
      { path: 'resources/:id', name: 'resource-detail', component: ResourceDetailPage, meta: { requiresAuth: true } },
      { path: 'user/me', name: 'user-me', component: ProfilePage, meta: { requiresAuth: true } },
      {
        path: 'resources/upload',
        name: 'resource-upload',
        component: UploadPage,
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
        component: AdminResources,
        meta: { requiresAuth: true, requiresAdmin: true },
      },
      {
        path: 'categories',
        name: 'admin-categories',
        component: AdminCategories,
        meta: { requiresAuth: true, requiresAdmin: true },
      },
      {
        path: 'recommendations',
        name: 'admin-recommendations',
        component: AdminRecommendations,
        meta: { requiresAuth: true, requiresAdmin: true },
      },
    ],
  },
  { path: '/:pathMatch(.*)*', name: 'not-found', component: NotFoundPage },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  // SPA navigations otherwise keep the previous page's scroll offset. Always
  // start a new page at the top; back/forward restores the saved position.
  scrollBehavior(to, _from, savedPosition) {
    if (savedPosition) return savedPosition
    if (to.hash) return { el: to.hash, top: 16 }
    return { top: 0, left: 0 }
  },
})

// One loading screen per navigation. The token makes redirects safe: the
// failed navigation's afterEach cannot end the redirect that replaced it.
const navigationTokens = new WeakMap<object, number>()

router.beforeEach(async (to) => {
  navigationTokens.set(to, pageLoading.startNavigation())
  const auth = useAuthStore()
  return resolveGuard(to, auth)
})

router.afterEach((to) => {
  pageLoading.endNavigation(navigationTokens.get(to))
  navigationTokens.delete(to)
})

router.onError(() => {
  pageLoading.endNavigation()
})

export default router
