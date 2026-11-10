export const reportsNavItems = [
  {
    titleKey: 'globals.terms.overview',
    href: '/reports/overview',
    permission: 'reports:manage',
    icon: 'BarChart3'
  },
  {
    titleKey: 'globals.terms.issues',
    href: '/reports/issues',
    permission: 'reports:manage',
    icon: 'LineChart'
  }
]

export const adminNavItems = [
  {
    titleKey: 'globals.terms.workspace',
    icon: 'Settings',
    children: [
      {
        titleKey: 'globals.terms.general',
        href: '/admin/general',
        permission: 'general_settings:manage',
        icon: 'Settings'
      },
      {
        titleKey: 'globals.terms.businessHour',
        href: '/admin/business-hours',
        createRouteName: 'new-business-hours',
        permission: 'business_hours:manage',
        isTitleKeyPlural: true,
        icon: 'Clock'
      },
      {
        titleKey: 'globals.terms.slaPolicy',
        href: '/admin/sla',
        createRouteName: 'new-sla',
        permission: 'sla:manage',
        isTitleKeyPlural: true,
        icon: 'Timer'
      }
    ]
  },
  {
    titleKey: 'globals.terms.helpCenter',
    badge: 'BETA',
    icon: 'BookOpen',
    children: [
      {
        titleKey: 'globals.terms.helpCenter',
        href: '/admin/help-center',
        permission: 'help_center:manage',
        icon: 'BookOpen'
      }
    ]
  },
  {
    titleKey: 'admin.ai.title',
    badge: 'BETA',
    icon: 'Sparkles',
    children: [
      {
        titleKey: 'globals.terms.provider',
        href: '/admin/ai/providers',
        permission: 'ai:manage',
        isTitleKeyPlural: true,
        icon: 'Sparkles'
      },
      {
        titleKey: 'admin.ai.snippets',
        href: '/admin/ai/snippets',
        permission: 'ai:manage',
        isTitleKeyPlural: true,
        icon: 'NotebookText'
      },
      {
        titleKey: 'admin.ai.editorPrompts',
        href: '/admin/ai/editor-prompts',
        permission: 'ai:manage',
        isTitleKeyPlural: true,
        icon: 'WandSparkles'
      },
      {
        titleKey: 'admin.ai.suggestions',
        href: '/admin/ai/suggestions',
        permission: 'ai:manage',
        isTitleKeyPlural: true,
        icon: 'Lightbulb'
      },
      {
        titleKey: 'admin.ai.tools',
        href: '/admin/ai/tools',
        createRouteName: 'new-ai-tool',
        permission: 'ai:manage',
        isTitleKeyPlural: true,
        icon: 'Wrench'
      },
      {
        titleKey: 'admin.ai.assistants',
        href: '/admin/ai/assistants',
        createRouteName: 'new-ai-assistant',
        permission: 'ai:manage',
        isTitleKeyPlural: true,
        icon: 'Bot'
      }
    ]
  },
  {
    titleKey: 'globals.terms.channel',
    isTitleKeyPlural: true,
    icon: 'Inbox',
    children: [
      {
        titleKey: 'globals.terms.inbox',
        href: '/admin/inboxes',
        createRouteName: 'new-inbox',
        permission: 'inboxes:manage',
        isTitleKeyPlural: true,
        icon: 'Inbox'
      },
      {
        titleKey: 'admin.whatsappTemplates.title',
        href: '/admin/whatsapp/templates',
        permission: 'inboxes:manage',
        icon: 'WhatsApp'
      }
    ]
  },
  {
    titleKey: 'globals.terms.conversation',
    isTitleKeyPlural: true,
    icon: 'MessageSquare',
    children: [
      {
        titleKey: 'globals.terms.status',
        href: '/admin/conversations/statuses',
        permission: 'status:manage',
        isTitleKeyPlural: true,
        icon: 'CircleDot'
      },
      {
        titleKey: 'globals.terms.tag',
        href: '/admin/conversations/tags',
        permission: 'tags:manage',
        isTitleKeyPlural: true,
        icon: 'Tag'
      },
      {
        titleKey: 'globals.terms.customAttribute',
        href: '/admin/custom-attributes',
        permission: 'custom_attributes:manage',
        isTitleKeyPlural: true,
        icon: 'SlidersHorizontal'
      },
      {
        titleKey: 'globals.terms.sharedView',
        href: '/admin/conversations/shared-views',
        createRouteName: 'new-shared-view',
        permission: 'shared_views:manage',
        isTitleKeyPlural: true,
        icon: 'Eye'
      }
    ]
  },
  {
    titleKey: 'globals.terms.productivity',
    icon: 'Zap',
    children: [
      {
        titleKey: 'globals.terms.macro',
        href: '/admin/conversations/macros',
        createRouteName: 'new-macro',
        permission: 'macros:manage',
        isTitleKeyPlural: true,
        icon: 'Zap'
      },
      {
        titleKey: 'globals.terms.automation',
        href: '/admin/automations',
        createRouteName: 'new-automation',
        permission: 'automations:manage',
        isTitleKeyPlural: true,
        icon: 'Workflow'
      }
    ]
  },
  {
    titleKey: 'globals.terms.teammate',
    isTitleKeyPlural: true,
    icon: 'UsersRound',
    children: [
      {
        titleKey: 'globals.terms.agent',
        href: '/admin/teams/agents',
        createRouteName: 'new-agent',
        permission: 'users:manage',
        isTitleKeyPlural: true,
        icon: 'UserRound'
      },
      {
        titleKey: 'globals.terms.team',
        href: '/admin/teams/teams',
        createRouteName: 'new-team',
        permission: 'teams:manage',
        isTitleKeyPlural: true,
        icon: 'UsersRound'
      },
      {
        titleKey: 'globals.terms.role',
        href: '/admin/teams/roles',
        createRouteName: 'new-role',
        permission: 'roles:manage',
        isTitleKeyPlural: true,
        icon: 'Shield'
      },
      {
        titleKey: 'globals.terms.activityLog',
        href: '/admin/teams/activity-log',
        permission: 'activity_logs:manage',
        isTitleKeyPlural: true,
        icon: 'ScrollText'
      }
    ]
  },
  {
    titleKey: 'globals.terms.notification',
    isTitleKeyPlural: true,
    icon: 'Bell',
    children: [
      {
        titleKey: 'globals.terms.email',
        href: '/admin/notification',
        permission: 'notification_settings:manage',
        icon: 'Mail'
      },
      {
        titleKey: 'globals.terms.template',
        href: '/admin/templates',
        createRouteName: 'new-template',
        permission: 'templates:manage',
        isTitleKeyPlural: true,
        icon: 'FileText'
      }
    ]
  },
  {
    titleKey: 'globals.terms.security',
    icon: 'Shield',
    children: [
      {
        titleKey: 'globals.terms.sso',
        href: '/admin/sso',
        createRouteName: 'new-sso',
        permission: 'oidc:manage',
        icon: 'KeyRound'
      }
    ]
  },
  {
    titleKey: 'globals.terms.integration',
    isTitleKeyPlural: true,
    icon: 'Webhook',
    children: [
      {
        titleKey: 'globals.terms.webhook',
        href: '/admin/webhooks',
        createRouteName: 'new-webhook',
        permission: 'webhooks:manage',
        isTitleKeyPlural: true,
        icon: 'Webhook'
      },
      {
        titleKey: 'globals.terms.contextLink',
        href: '/admin/context-links',
        createRouteName: 'new-context-link',
        permission: 'context_links:manage',
        isTitleKeyPlural: true,
        icon: 'Link'
      }
    ]
  }
]

export const accountNavItems = [
  {
    titleKey: 'globals.terms.profile',
    href: '/account/profile',
    icon: 'CircleUser'
  },
  {
    titleKey: 'globals.terms.notification',
    href: '/account/notifications',
    isTitleKeyPlural: true,
    icon: 'Bell'
  }
]

export const contactNavItems = [
  {
    titleKey: 'globals.terms.contact',
    allLabelKey: 'contact.allContacts',
    href: '/contacts',
    isTitleKeyPlural: true,
    icon: 'Contact'
  }
]
