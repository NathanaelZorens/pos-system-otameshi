import { createRouter, createWebHistory } from 'vue-router'
import LoginView from './views/LoginView.vue'
import TablesView from './views/TablesView.vue'
import DraftOrderView from './views/DraftOrderView.vue'
import OrderView from './views/OrderView.vue'
import ReceiptView from './views/ReceiptView.vue'
import MenuAdminView from './views/MenuAdminView.vue'
import ReportsView from './views/ReportsView.vue'
import CancelledOpsView from './views/CancelledOpsView.vue'
import GuestOrderView from './views/GuestOrderView.vue'
import GuestReceiptView from './views/GuestReceiptView.vue'
import DraftTakeoutView from './views/DraftTakeoutView.vue'
import TakeoutOrderView from './views/TakeoutOrderView.vue'
import TakeoutQueueView from './views/TakeoutQueueView.vue'
import MemberLoginView from './views/MemberLoginView.vue'
import MemberHomeView from './views/MemberHomeView.vue'
import MemberTakeoutView from './views/MemberTakeoutView.vue'
import GuestTakeoutIntroView from './views/GuestTakeoutIntroView.vue'
import GuestTakeoutReceiptView from './views/GuestTakeoutReceiptView.vue'
import MemberReceiptView from './views/MemberReceiptView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: LoginView },
    { path: '/', redirect: '/tables' },
    { path: '/tables', component: TablesView, meta: { auth: true } },
    { path: '/tables/:tableId/order', component: DraftOrderView, meta: { auth: true } },
    { path: '/orders/:id', component: OrderView, meta: { auth: true } },
    { path: '/receipt/:id', component: ReceiptView, meta: { auth: true } },
    { path: '/takeout-queue', component: TakeoutQueueView, meta: { auth: true } },
    { path: '/takeout-queue/new', component: DraftTakeoutView, meta: { auth: true } },
    { path: '/admin/menu', component: MenuAdminView, meta: { auth: true, role: 'admin' } },
    { path: '/admin/reports', component: ReportsView, meta: { auth: true, role: 'admin' } },
    { path: '/admin/cancelled', component: CancelledOpsView, meta: { auth: true, role: 'admin' } },
    { path: '/t/:token', component: GuestOrderView, meta: { guest: true } },
    { path: '/t/:token/receipt/:orderId', component: GuestReceiptView, meta: { guest: true } },
    { path: '/takeout', component: GuestTakeoutIntroView, meta: { guest: true } },
    { path: '/takeout/order', component: TakeoutOrderView, meta: { guest: true } },
    { path: '/takeout/receipt/:orderId', component: GuestTakeoutReceiptView, meta: { guest: true } },
    { path: '/member/login', component: MemberLoginView, meta: { guest: true } },
    { path: '/member', component: MemberHomeView, meta: { member: true } },
    { path: '/member/takeout', component: MemberTakeoutView, meta: { member: true } },
    { path: '/member/receipt/:orderId', component: MemberReceiptView, meta: { member: true } },
  ],
})

router.beforeEach((to) => {
  const token = localStorage.getItem('token')
  const user = JSON.parse(localStorage.getItem('user') || 'null')
  const memberToken = localStorage.getItem('memberToken')
  if (to.meta.guest) return true
  if (to.meta.member) {
    if (!memberToken) return '/member/login'
    return true
  }
  if (to.meta.auth && !token) return '/login'
  if (to.meta.role && user?.role !== to.meta.role) return '/tables'
  if (to.path === '/login' && token) return '/tables'
  if (to.path === '/member/login' && memberToken) return '/member'
  return true
})

export default router
