import { defineConfig } from 'vitepress';

export default defineConfig({
  title: "Documentation",
  lang: 'en-US',
  base: '/go-store',
  lastUpdated: true,
  cleanUrls: true,

  head: [
    ['link', { rel: 'icon', href: '/favicon.ico' }]
  ],

  themeConfig: {
    nav: [
    {
        "text": "Getting Started",
        "link": "/getting-started"
    },
    {
        "text": "Core Operations",
        "link": "/core-operations"
    },
    {
        "text": "Task-Based Guide",
        "link": "/task-based-guide"
    },
    {
        "text": "Advanced Usage",
        "link": "/advanced-usage"
    },
    {
        "text": "Project Architecture",
        "link": "/project-architecture"
    },
    {
        "text": "Development & Contributing",
        "link": "/development-contributing"
    },
    {
        "text": "Problem Solving",
        "link": "/problem-solving"
    },
    {
        "text": "Reference",
        "link": "/reference/system"
    }
],
    sidebar: [
    {
        "text": "Getting Started",
        "items": [
            {
                "text": "Getting Started",
                "link": "/getting-started#getting-started"
            },
            {
                "text": "Prerequisites",
                "link": "/getting-started#prerequisites"
            },
            {
                "text": "Installation Steps",
                "link": "/getting-started#installation-steps"
            },
            {
                "text": "Verification",
                "link": "/getting-started#verification"
            },
            {
                "text": "First Tasks: Basic Document Operations",
                "link": "/getting-started#first-tasks-basic-document-operations"
            }
        ]
    },
    {
        "text": "Core Operations",
        "items": [
            {
                "text": "Core Operations",
                "link": "/core-operations#core-operations"
            },
            {
                "text": "Initializing the Store",
                "link": "/core-operations#initializing-the-store"
            },
            {
                "text": "Inserting Documents",
                "link": "/core-operations#inserting-documents"
            },
            {
                "text": "Retrieving Documents",
                "link": "/core-operations#retrieving-documents"
            },
            {
                "text": "Updating Documents",
                "link": "/core-operations#updating-documents"
            },
            {
                "text": "Deleting Documents",
                "link": "/core-operations#deleting-documents"
            }
        ]
    },
    {
        "text": "Task-Based Guide",
        "items": [
            {
                "text": "Task-Based Guide",
                "link": "/task-based-guide#task-based-guide"
            },
            {
                "text": "Indexing and Querying Data",
                "link": "/task-based-guide#indexing-and-querying-data"
            },
            {
                "text": "Streaming Documents",
                "link": "/task-based-guide#streaming-documents"
            }
        ]
    },
    {
        "text": "Advanced Usage",
        "items": [
            {
                "text": "Advanced Usage",
                "link": "/advanced-usage#advanced-usage"
            },
            {
                "text": "Concurrent Operations",
                "link": "/advanced-usage#concurrent-operations"
            },
            {
                "text": "Understanding Concurrency Control and Memory Management",
                "link": "/advanced-usage#understanding-concurrency-control-and-memory-management"
            },
            {
                "text": "Importance of `Close()` Method",
                "link": "/advanced-usage#importance-of-close-method"
            },
            {
                "text": "Data Copying Behavior",
                "link": "/advanced-usage#data-copying-behavior"
            }
        ]
    },
    {
        "text": "Project Architecture",
        "items": [
            {
                "text": "Project Architecture",
                "link": "/project-architecture#project-architecture"
            },
            {
                "text": "Data Flow",
                "link": "/project-architecture#data-flow"
            }
        ]
    },
    {
        "text": "Development & Contributing",
        "items": [
            {
                "text": "Development & Contributing",
                "link": "/development-contributing#development-contributing"
            },
            {
                "text": "Development Setup",
                "link": "/development-contributing#development-setup"
            },
            {
                "text": "Scripts",
                "link": "/development-contributing#scripts"
            },
            {
                "text": "Testing",
                "link": "/development-contributing#testing"
            },
            {
                "text": "Contributing Guidelines",
                "link": "/development-contributing#contributing-guidelines"
            },
            {
                "text": "Issue Reporting",
                "link": "/development-contributing#issue-reporting"
            }
        ]
    },
    {
        "text": "Problem Solving",
        "items": [
            {
                "text": "Problem Solving",
                "link": "/problem-solving#problem-solving"
            },
            {
                "text": "Troubleshooting Common Errors",
                "link": "/problem-solving#troubleshooting-common-errors"
            },
            {
                "text": "FAQ (Frequently Asked Questions)",
                "link": "/problem-solving#faq-frequently-asked-questions"
            }
        ]
    },
    {
        "text": "Types",
        "items": [
            {
                "text": "DocumentResult",
                "link": "/reference/types#documentresult"
            }
        ]
    },
    {
        "text": "Interfaces",
        "items": [
            {
                "text": "Document",
                "link": "/reference/interfaces#document"
            },
            {
                "text": "DocumentStream",
                "link": "/reference/interfaces#documentstream"
            }
        ]
    },
    {
        "text": "Examples",
        "items": [
            {
                "text": "Basic Document CRUD",
                "link": "/reference/patterns#basic-document-crud"
            },
            {
                "text": "Indexed Lookup",
                "link": "/reference/patterns#indexed-lookup"
            },
            {
                "text": "Streaming Documents",
                "link": "/reference/patterns#streaming-documents"
            },
            {
                "text": "Concurrent Update Pattern",
                "link": "/reference/patterns#concurrent-update-pattern"
            }
        ]
    },
    {
        "text": "Reference",
        "items": [
            {
                "text": "System Overview",
                "link": "/reference/system"
            },
            {
                "text": "Dependencies",
                "link": "/reference/dependencies"
            },
            {
                "text": "Integration",
                "link": "/reference/integration"
            },
            {
                "text": "Methods",
                "link": "/reference/methods"
            },
            {
                "text": "Error Reference",
                "link": "/reference/errors"
            }
        ]
    }
],

    socialLinks: [
      // { icon: 'github', link: '[https://github.com/asaidimu/$](https://github.com/asaidimu/$){data.reference.system.name.toLowerCase()}' }
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2023-present go-store'
    }
  }
});
