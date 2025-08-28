/**
 * GSAP Animation Library for Task Management UI
 * Provides smooth, performance-optimized animations for task management interfaces
 */

// Check if GSAP is loaded
const GSAPLoaded = typeof gsap !== 'undefined';

if (!GSAPLoaded) {
    console.warn('GSAP not loaded. Animations will be disabled.');
}

const TaskAnimations = {
    // Configuration
    config: {
        defaultDuration: 0.3,
        defaultEase: "power2.out",
        backEase: "back.out(1.7)",
        bounceEase: "bounce.out"
    },

    // Task Card Animations
    taskAppear(element, options = {}) {
        if (!GSAPLoaded) return Promise.resolve();

        const duration = options.duration || 0.4;
        const delay = options.delay || 0;

        return gsap.fromTo(element, 
            { 
                scale: 0, 
                opacity: 0, 
                y: 20,
                rotationX: -90
            },
            { 
                scale: 1, 
                opacity: 1, 
                y: 0,
                rotationX: 0,
                duration: duration,
                delay: delay,
                ease: this.config.backEase,
                transformOrigin: "center top"
            }
        );
    },
    
    taskDisappear(element, options = {}) {
        if (!GSAPLoaded) return Promise.resolve();

        const duration = options.duration || 0.3;

        return gsap.to(element, {
            scale: 0,
            opacity: 0,
            y: -20,
            rotationX: 90,
            duration: duration,
            ease: "back.in(1.7)",
            transformOrigin: "center top"
        });
    },
    
    taskHover(element, options = {}) {
        if (!GSAPLoaded) return Promise.resolve();

        const duration = options.duration || 0.2;

        return gsap.to(element, {
            y: -2,
            scale: 1.02,
            boxShadow: "0 10px 25px -5px rgba(0, 0, 0, 0.15)",
            duration: duration,
            ease: this.config.defaultEase
        });
    },
    
    taskUnhover(element, options = {}) {
        if (!GSAPLoaded) return Promise.resolve();

        const duration = options.duration || 0.2;

        return gsap.to(element, {
            y: 0,
            scale: 1,
            boxShadow: "0 4px 6px -1px rgba(0, 0, 0, 0.1)",
            duration: duration,
            ease: this.config.defaultEase
        });
    },

    taskPulse(element, options = {}) {
        if (!GSAPLoaded) return Promise.resolve();

        const intensity = options.intensity || 1.05;

        return gsap.to(element, {
            scale: intensity,
            duration: 0.1,
            ease: "power2.out",
            yoyo: true,
            repeat: 1
        });
    },

    // Drag and Drop Animations
    taskDragStart(element, options = {}) {
        if (!GSAPLoaded) return Promise.resolve();

        return gsap.to(element, {
            scale: 1.05,
            rotation: 5,
            zIndex: 1000,
            boxShadow: "0 25px 50px -12px rgba(0, 0, 0, 0.25)",
            duration: 0.2,
            ease: this.config.defaultEase
        });
    },

    taskDragEnd(element, options = {}) {
        if (!GSAPLoaded) return Promise.resolve();

        return gsap.to(element, {
            x: 0,
            y: 0,
            scale: 1,
            rotation: 0,
            zIndex: "auto",
            boxShadow: "0 4px 6px -1px rgba(0, 0, 0, 0.1)",
            duration: 0.3,
            ease: this.config.defaultEase
        });
    },

    taskMoveToColumn(element, targetColumn, options = {}) {
        if (!GSAPLoaded) return Promise.resolve();

        const duration = options.duration || 0.4;

        return gsap.timeline()
            .to(element, {
                scale: 0.8,
                opacity: 0.7,
                duration: 0.2
            })
            .call(() => {
                if (targetColumn) {
                    targetColumn.appendChild(element);
                }
            })
            .fromTo(element,
                { scale: 0.8, opacity: 0.7 },
                { scale: 1, opacity: 1, duration: 0.3, ease: this.config.backEase }
            );
    },
    
    // Column Animations
    columnHighlight(element, options = {}) {
        if (!GSAPLoaded) return Promise.resolve();

        const color = options.color || "#f0f9ff";

        return gsap.to(element, {
            backgroundColor: color,
            scale: 1.02,
            duration: 0.2,
            ease: this.config.defaultEase
        });
    },
    
    columnUnhighlight(element, options = {}) {
        if (!GSAPLoaded) return Promise.resolve();

        return gsap.to(element, {
            backgroundColor: "#ffffff",
            scale: 1,
            duration: 0.2,
            ease: this.config.defaultEase
        });
    },

    columnShake(element, options = {}) {
        if (!GSAPLoaded) return Promise.resolve();

        const intensity = options.intensity || 10;

        return gsap.to(element, {
            x: intensity,
            duration: 0.1,
            ease: "power2.inOut",
            yoyo: true,
            repeat: 5
        }).then(() => {
            return gsap.set(element, { x: 0 });
        });
    },
    
    // Modal Animations
    modalShow(element, options = {}) {
        if (!GSAPLoaded) {
            element.style.display = 'flex';
            return Promise.resolve();
        }

        const backdrop = element.querySelector('.modal-backdrop');
        const content = element.querySelector('.modal-content');
        
        gsap.set(element, { display: 'flex' });
        
        const tl = gsap.timeline();
        
        if (backdrop) {
            tl.fromTo(backdrop, 
                { opacity: 0 },
                { opacity: 1, duration: 0.3 }
            );
        }
        
        if (content) {
            tl.fromTo(content,
                { scale: 0.9, opacity: 0, y: 20 },
                { scale: 1, opacity: 1, y: 0, duration: 0.3, ease: this.config.backEase },
                backdrop ? "<0.1" : 0
            );
        }
        
        return tl;
    },
    
    modalHide(element, options = {}) {
        if (!GSAPLoaded) {
            element.style.display = 'none';
            return Promise.resolve();
        }

        const backdrop = element.querySelector('.modal-backdrop');
        const content = element.querySelector('.modal-content');
        
        const tl = gsap.timeline();
        
        if (content) {
            tl.to(content, { 
                scale: 0.9, 
                opacity: 0, 
                y: 20, 
                duration: 0.2,
                ease: "back.in(1.7)" 
            });
        }
        
        if (backdrop) {
            tl.to(backdrop, { 
                opacity: 0, 
                duration: 0.2 
            }, content ? "<" : 0);
        }
        
        tl.set(element, { display: 'none' });
        
        return tl;
    },
    
    // Loading Animations
    loadingSpinner(element, options = {}) {
        if (!GSAPLoaded) return Promise.resolve();

        return gsap.to(element, {
            rotation: 360,
            duration: 1,
            repeat: -1,
            ease: "none"
        });
    },

    loadingPulse(element, options = {}) {
        if (!GSAPLoaded) return Promise.resolve();

        return gsap.to(element, {
            opacity: 0.5,
            duration: 1,
            yoyo: true,
            repeat: -1,
            ease: "power2.inOut"
        });
    },

    loadingDots(elements, options = {}) {
        if (!GSAPLoaded || !elements.length) return Promise.resolve();

        const tl = gsap.timeline({ repeat: -1 });
        
        elements.forEach((element, index) => {
            tl.to(element, {
                y: -10,
                duration: 0.4,
                ease: "power2.out",
                yoyo: true,
                repeat: 1
            }, index * 0.1);
        });

        return tl;
    },
    
    // Notification Animations
    notificationShow(element, options = {}) {
        if (!GSAPLoaded) return Promise.resolve();

        const fromX = options.fromRight ? 300 : -300;

        return gsap.fromTo(element,
            { x: fromX, opacity: 0 },
            { x: 0, opacity: 1, duration: 0.4, ease: this.config.backEase }
        );
    },
    
    notificationHide(element, options = {}) {
        if (!GSAPLoaded) return Promise.resolve();

        const toX = options.toRight ? 300 : -300;

        return gsap.to(element, {
            x: toX,
            opacity: 0,
            duration: 0.3,
            ease: "back.in(1.7)"
        });
    },

    // Form Animations
    formFieldError(element, options = {}) {
        if (!GSAPLoaded) return Promise.resolve();

        return gsap.timeline()
            .to(element, {
                x: 10,
                duration: 0.1,
                ease: "power2.out"
            })
            .to(element, {
                x: -10,
                duration: 0.1,
                ease: "power2.out"
            })
            .to(element, {
                x: 5,
                duration: 0.1,
                ease: "power2.out"
            })
            .to(element, {
                x: 0,
                duration: 0.1,
                ease: "power2.out"
            });
    },

    formFieldSuccess(element, options = {}) {
        if (!GSAPLoaded) return Promise.resolve();

        return gsap.to(element, {
            backgroundColor: "#10b981",
            color: "#ffffff",
            duration: 0.2,
            ease: "power2.out",
            yoyo: true,
            repeat: 1
        });
    },

    // Page Transition Animations
    pageEnter(element, options = {}) {
        if (!GSAPLoaded) return Promise.resolve();

        const duration = options.duration || 0.6;
        const stagger = options.stagger || 0.1;

        // Animate page content
        const children = element.children;
        
        return gsap.fromTo(children,
            { opacity: 0, y: 30 },
            { 
                opacity: 1, 
                y: 0, 
                duration: duration,
                stagger: stagger,
                ease: this.config.defaultEase 
            }
        );
    },

    pageExit(element, options = {}) {
        if (!GSAPLoaded) return Promise.resolve();

        const duration = options.duration || 0.4;

        return gsap.to(element, {
            opacity: 0,
            y: -20,
            duration: duration,
            ease: "power2.in"
        });
    },

    // Utility Functions
    staggerItems(elements, animation, options = {}) {
        if (!GSAPLoaded || !elements.length) return Promise.resolve();

        const stagger = options.stagger || 0.1;
        const tl = gsap.timeline();

        elements.forEach((element, index) => {
            tl.add(animation(element), index * stagger);
        });

        return tl;
    },

    // Cleanup function to kill all animations
    killAll() {
        if (GSAPLoaded) {
            gsap.killTweensOf("*");
        }
    },

    // Performance monitoring
    performance: {
        enabled: false,
        
        enable() {
            this.enabled = true;
            if (GSAPLoaded) {
                gsap.ticker.fps(60);
            }
        },
        
        disable() {
            this.enabled = false;
        },
        
        log(message) {
            if (this.enabled) {
                console.log(`[TaskAnimations] ${message}`);
            }
        }
    }
};

// Initialize animations on page load
document.addEventListener('DOMContentLoaded', function() {
    console.log('Initializing TaskAnimations...');
    
    // Enable performance monitoring in development
    if (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1') {
        TaskAnimations.performance.enable();
    }
    
    // Animate page entrance
    const pageContent = document.querySelector('.page-content, main, .container');
    if (pageContent) {
        TaskAnimations.pageEnter(pageContent);
    }
    
    // Setup hover effects for existing task cards
    document.querySelectorAll('.task-card').forEach(card => {
        card.addEventListener('mouseenter', () => {
            TaskAnimations.taskHover(card);
        });
        
        card.addEventListener('mouseleave', () => {
            TaskAnimations.taskUnhover(card);
        });
    });

    // Setup loading spinners
    document.querySelectorAll('.loading-spinner').forEach(spinner => {
        TaskAnimations.loadingSpinner(spinner);
    });

    // Setup notification animations
    document.querySelectorAll('.notification').forEach(notification => {
        TaskAnimations.notificationShow(notification, { fromRight: true });
    });

    console.log('TaskAnimations initialized successfully');
});

// Export for global access
window.TaskAnimations = TaskAnimations;