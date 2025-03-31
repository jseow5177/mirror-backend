package handler

import (
	"cdp/config"
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"cdp/pkg/validator"
	"cdp/repo"
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"math/rand"
	"strings"
	"time"
)

const (
	targetLink            = "https://www.mirrorcdp.com/"
	dummyEmailHtmlEncoded = "JTNDIURPQ1RZUEUlMjBIVE1MJTIwUFVCTElDJTIwJTIyLSUyRiUyRlczQyUyRiUyRkRURCUyMFhIVE1MJTIwMS4wJTIwVHJhbnNpdGlvbmFsJTIwJTJGJTJGRU4lMjIlMjAlMjJodHRwJTNBJTJGJTJGd3d3LnczLm9yZyUyRlRSJTJGeGh0bWwxJTJGRFREJTJGeGh0bWwxLXRyYW5zaXRpb25hbC5kdGQlMjIlM0UlMEElM0NodG1sJTIweG1sbnMlM0QlMjJodHRwJTNBJTJGJTJGd3d3LnczLm9yZyUyRjE5OTklMkZ4aHRtbCUyMiUyMHhtbG5zJTNBdiUzRCUyMnVybiUzQXNjaGVtYXMtbWljcm9zb2Z0LWNvbSUzQXZtbCUyMiUyMHhtbG5zJTNBbyUzRCUyMnVybiUzQXNjaGVtYXMtbWljcm9zb2Z0LWNvbSUzQW9mZmljZSUzQW9mZmljZSUyMiUzRSUwQSUzQ2hlYWQlM0UlMEElM0MhLS0lNUJpZiUyMGd0ZSUyMG1zbyUyMDklNUQlM0UlMEElM0N4bWwlM0UlMEElMjAlMjAlM0NvJTNBT2ZmaWNlRG9jdW1lbnRTZXR0aW5ncyUzRSUwQSUyMCUyMCUyMCUyMCUzQ28lM0FBbGxvd1BORyUyRiUzRSUwQSUyMCUyMCUyMCUyMCUzQ28lM0FQaXhlbHNQZXJJbmNoJTNFOTYlM0MlMkZvJTNBUGl4ZWxzUGVySW5jaCUzRSUwQSUyMCUyMCUzQyUyRm8lM0FPZmZpY2VEb2N1bWVudFNldHRpbmdzJTNFJTBBJTNDJTJGeG1sJTNFJTBBJTNDISU1QmVuZGlmJTVELS0lM0UlMEElMjAlMjAlM0NtZXRhJTIwaHR0cC1lcXVpdiUzRCUyMkNvbnRlbnQtVHlwZSUyMiUyMGNvbnRlbnQlM0QlMjJ0ZXh0JTJGaHRtbCUzQiUyMGNoYXJzZXQlM0RVVEYtOCUyMiUzRSUwQSUyMCUyMCUzQ21ldGElMjBuYW1lJTNEJTIydmlld3BvcnQlMjIlMjBjb250ZW50JTNEJTIyd2lkdGglM0RkZXZpY2Utd2lkdGglMkMlMjBpbml0aWFsLXNjYWxlJTNEMS4wJTIyJTNFJTBBJTIwJTIwJTNDbWV0YSUyMG5hbWUlM0QlMjJ4LWFwcGxlLWRpc2FibGUtbWVzc2FnZS1yZWZvcm1hdHRpbmclMjIlM0UlMEElMjAlMjAlM0MhLS0lNUJpZiUyMCFtc28lNUQlM0UlM0MhLS0lM0UlM0NtZXRhJTIwaHR0cC1lcXVpdiUzRCUyMlgtVUEtQ29tcGF0aWJsZSUyMiUyMGNvbnRlbnQlM0QlMjJJRSUzRGVkZ2UlMjIlM0UlM0MhLS0lM0MhJTVCZW5kaWYlNUQtLSUzRSUwQSUyMCUyMCUzQ3RpdGxlJTNFJTNDJTJGdGl0bGUlM0UlMEElMjAlMjAlMEElMjAlMjAlMjAlMjAlM0NzdHlsZSUyMHR5cGUlM0QlMjJ0ZXh0JTJGY3NzJTIyJTNFJTBBJTIwJTIwJTIwJTIwJTIwJTIwJTBBJTIwJTIwJTIwJTIwJTIwJTIwJTQwbWVkaWElMjBvbmx5JTIwc2NyZWVuJTIwYW5kJTIwKG1pbi13aWR0aCUzQSUyMDUyMHB4KSUyMCU3QiUwQSUyMCUyMCUyMCUyMCUyMCUyMCUyMCUyMC51LXJvdyUyMCU3QiUwQSUyMCUyMCUyMCUyMCUyMCUyMCUyMCUyMCUyMCUyMHdpZHRoJTNBJTIwNTAwcHglMjAhaW1wb3J0YW50JTNCJTBBJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTdEJTBBJTBBJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwLnUtcm93JTIwLnUtY29sJTIwJTdCJTBBJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwdmVydGljYWwtYWxpZ24lM0ElMjB0b3AlM0IlMEElMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAlN0QlMEElMEElMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAlMEElMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAudS1yb3clMjAudS1jb2wtMTAwJTIwJTdCJTBBJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwd2lkdGglM0ElMjA1MDBweCUyMCFpbXBvcnRhbnQlM0IlMEElMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAlN0QlMEElMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAlMEElMjAlMjAlMjAlMjAlMjAlMjAlN0QlMEElMEElMjAlMjAlMjAlMjAlMjAlMjAlNDBtZWRpYSUyMG9ubHklMjBzY3JlZW4lMjBhbmQlMjAobWF4LXdpZHRoJTNBJTIwNTIwcHgpJTIwJTdCJTBBJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwLnUtcm93LWNvbnRhaW5lciUyMCU3QiUwQSUyMCUyMCUyMCUyMCUyMCUyMCUyMCUyMCUyMCUyMG1heC13aWR0aCUzQSUyMDEwMCUyNSUyMCFpbXBvcnRhbnQlM0IlMEElMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjBwYWRkaW5nLWxlZnQlM0ElMjAwcHglMjAhaW1wb3J0YW50JTNCJTBBJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwcGFkZGluZy1yaWdodCUzQSUyMDBweCUyMCFpbXBvcnRhbnQlM0IlMEElMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAlN0QlMEElMEElMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAudS1yb3clMjAlN0IlMEElMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjB3aWR0aCUzQSUyMDEwMCUyNSUyMCFpbXBvcnRhbnQlM0IlMEElMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAlN0QlMEElMEElMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAudS1yb3clMjAudS1jb2wlMjAlN0IlMEElMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjBkaXNwbGF5JTNBJTIwYmxvY2slMjAhaW1wb3J0YW50JTNCJTBBJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwd2lkdGglM0ElMjAxMDAlMjUlMjAhaW1wb3J0YW50JTNCJTBBJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwbWluLXdpZHRoJTNBJTIwMzIwcHglMjAhaW1wb3J0YW50JTNCJTBBJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwbWF4LXdpZHRoJTNBJTIwMTAwJTI1JTIwIWltcG9ydGFudCUzQiUwQSUyMCUyMCUyMCUyMCUyMCUyMCUyMCUyMCU3RCUwQSUwQSUyMCUyMCUyMCUyMCUyMCUyMCUyMCUyMC51LXJvdyUyMC51LWNvbCUyMCUzRSUyMGRpdiUyMCU3QiUwQSUyMCUyMCUyMCUyMCUyMCUyMCUyMCUyMCUyMCUyMG1hcmdpbiUzQSUyMDAlMjBhdXRvJTNCJTBBJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTdEJTBBJTBBJTBBJTdEJTBBJTIwJTIwJTIwJTIwJTBBYm9keSU3Qm1hcmdpbiUzQTAlM0JwYWRkaW5nJTNBMCU3RHRhYmxlJTJDdGQlMkN0ciU3QmJvcmRlci1jb2xsYXBzZSUzQWNvbGxhcHNlJTNCdmVydGljYWwtYWxpZ24lM0F0b3AlN0RwJTdCbWFyZ2luJTNBMCU3RC5pZS1jb250YWluZXIlMjB0YWJsZSUyQy5tc28tY29udGFpbmVyJTIwdGFibGUlN0J0YWJsZS1sYXlvdXQlM0FmaXhlZCU3RColN0JsaW5lLWhlaWdodCUzQWluaGVyaXQlN0RhJTVCeC1hcHBsZS1kYXRhLWRldGVjdG9ycyUzRHRydWUlNUQlN0Jjb2xvciUzQWluaGVyaXQhaW1wb3J0YW50JTNCdGV4dC1kZWNvcmF0aW9uJTNBbm9uZSFpbXBvcnRhbnQlN0QlMEElMEElMEF0YWJsZSUyQyUyMHRkJTIwJTdCJTIwY29sb3IlM0ElMjAlMjMwMDAwMDAlM0IlMjAlN0QlMjAlMjN1X2JvZHklMjBhJTIwJTdCJTIwY29sb3IlM0ElMjAlMjMwMDAwZWUlM0IlMjB0ZXh0LWRlY29yYXRpb24lM0ElMjB1bmRlcmxpbmUlM0IlMjAlN0QlMEElMjAlMjAlMjAlMjAlM0MlMkZzdHlsZSUzRSUwQSUyMCUyMCUwQSUyMCUyMCUwQSUwQSUzQyUyRmhlYWQlM0UlMEElMEElM0Nib2R5JTIwY2xhc3MlM0QlMjJjbGVhbi1ib2R5JTIwdV9ib2R5JTIyJTIwc3R5bGUlM0QlMjJtYXJnaW4lM0ElMjAwJTNCcGFkZGluZyUzQSUyMDAlM0Itd2Via2l0LXRleHQtc2l6ZS1hZGp1c3QlM0ElMjAxMDAlMjUlM0JiYWNrZ3JvdW5kLWNvbG9yJTNBJTIwJTIzRjdGOEY5JTNCY29sb3IlM0ElMjAlMjMwMDAwMDAlMjIlM0UlMEElMjAlMjAlM0MhLS0lNUJpZiUyMElFJTVEJTNFJTNDZGl2JTIwY2xhc3MlM0QlMjJpZS1jb250YWluZXIlMjIlM0UlM0MhJTVCZW5kaWYlNUQtLSUzRSUwQSUyMCUyMCUzQyEtLSU1QmlmJTIwbXNvJTVEJTNFJTNDZGl2JTIwY2xhc3MlM0QlMjJtc28tY29udGFpbmVyJTIyJTNFJTNDISU1QmVuZGlmJTVELS0lM0UlMEElMjAlMjAlM0N0YWJsZSUyMGlkJTNEJTIydV9ib2R5JTIyJTIwc3R5bGUlM0QlMjJib3JkZXItY29sbGFwc2UlM0ElMjBjb2xsYXBzZSUzQnRhYmxlLWxheW91dCUzQSUyMGZpeGVkJTNCYm9yZGVyLXNwYWNpbmclM0ElMjAwJTNCbXNvLXRhYmxlLWxzcGFjZSUzQSUyMDBwdCUzQm1zby10YWJsZS1yc3BhY2UlM0ElMjAwcHQlM0J2ZXJ0aWNhbC1hbGlnbiUzQSUyMHRvcCUzQm1pbi13aWR0aCUzQSUyMDMyMHB4JTNCTWFyZ2luJTNBJTIwMCUyMGF1dG8lM0JiYWNrZ3JvdW5kLWNvbG9yJTNBJTIwJTIzRjdGOEY5JTNCd2lkdGglM0ExMDAlMjUlMjIlMjBjZWxscGFkZGluZyUzRCUyMjAlMjIlMjBjZWxsc3BhY2luZyUzRCUyMjAlMjIlM0UlMEElMjAlMjAlM0N0Ym9keSUzRSUwQSUyMCUyMCUzQ3RyJTIwc3R5bGUlM0QlMjJ2ZXJ0aWNhbC1hbGlnbiUzQSUyMHRvcCUyMiUzRSUwQSUyMCUyMCUyMCUyMCUzQ3RkJTIwc3R5bGUlM0QlMjJ3b3JkLWJyZWFrJTNBJTIwYnJlYWstd29yZCUzQmJvcmRlci1jb2xsYXBzZSUzQSUyMGNvbGxhcHNlJTIwIWltcG9ydGFudCUzQnZlcnRpY2FsLWFsaWduJTNBJTIwdG9wJTIyJTNFJTBBJTIwJTIwJTIwJTIwJTNDIS0tJTVCaWYlMjAobXNvKSU3QyhJRSklNUQlM0UlM0N0YWJsZSUyMHdpZHRoJTNEJTIyMTAwJTI1JTIyJTIwY2VsbHBhZGRpbmclM0QlMjIwJTIyJTIwY2VsbHNwYWNpbmclM0QlMjIwJTIyJTIwYm9yZGVyJTNEJTIyMCUyMiUzRSUzQ3RyJTNFJTNDdGQlMjBhbGlnbiUzRCUyMmNlbnRlciUyMiUyMHN0eWxlJTNEJTIyYmFja2dyb3VuZC1jb2xvciUzQSUyMCUyM0Y3RjhGOSUzQiUyMiUzRSUzQyElNUJlbmRpZiU1RC0tJTNFJTBBJTIwJTIwJTIwJTIwJTBBJTIwJTIwJTBBJTIwJTIwJTBBJTNDZGl2JTIwY2xhc3MlM0QlMjJ1LXJvdy1jb250YWluZXIlMjIlMjBzdHlsZSUzRCUyMnBhZGRpbmclM0ElMjAwcHglM0JiYWNrZ3JvdW5kLWNvbG9yJTNBJTIwdHJhbnNwYXJlbnQlMjIlM0UlMEElMjAlMjAlM0NkaXYlMjBjbGFzcyUzRCUyMnUtcm93JTIyJTIwc3R5bGUlM0QlMjJtYXJnaW4lM0ElMjAwJTIwYXV0byUzQm1pbi13aWR0aCUzQSUyMDMyMHB4JTNCbWF4LXdpZHRoJTNBJTIwNTAwcHglM0JvdmVyZmxvdy13cmFwJTNBJTIwYnJlYWstd29yZCUzQndvcmQtd3JhcCUzQSUyMGJyZWFrLXdvcmQlM0J3b3JkLWJyZWFrJTNBJTIwYnJlYWstd29yZCUzQmJhY2tncm91bmQtY29sb3IlM0ElMjB0cmFuc3BhcmVudCUzQiUyMiUzRSUwQSUyMCUyMCUyMCUyMCUzQ2RpdiUyMHN0eWxlJTNEJTIyYm9yZGVyLWNvbGxhcHNlJTNBJTIwY29sbGFwc2UlM0JkaXNwbGF5JTNBJTIwdGFibGUlM0J3aWR0aCUzQSUyMDEwMCUyNSUzQmhlaWdodCUzQSUyMDEwMCUyNSUzQmJhY2tncm91bmQtY29sb3IlM0ElMjB0cmFuc3BhcmVudCUzQiUyMiUzRSUwQSUyMCUyMCUyMCUyMCUyMCUyMCUzQyEtLSU1QmlmJTIwKG1zbyklN0MoSUUpJTVEJTNFJTNDdGFibGUlMjB3aWR0aCUzRCUyMjEwMCUyNSUyMiUyMGNlbGxwYWRkaW5nJTNEJTIyMCUyMiUyMGNlbGxzcGFjaW5nJTNEJTIyMCUyMiUyMGJvcmRlciUzRCUyMjAlMjIlM0UlM0N0ciUzRSUzQ3RkJTIwc3R5bGUlM0QlMjJwYWRkaW5nJTNBJTIwMHB4JTNCYmFja2dyb3VuZC1jb2xvciUzQSUyMHRyYW5zcGFyZW50JTNCJTIyJTIwYWxpZ24lM0QlMjJjZW50ZXIlMjIlM0UlM0N0YWJsZSUyMGNlbGxwYWRkaW5nJTNEJTIyMCUyMiUyMGNlbGxzcGFjaW5nJTNEJTIyMCUyMiUyMGJvcmRlciUzRCUyMjAlMjIlMjBzdHlsZSUzRCUyMndpZHRoJTNBNTAwcHglM0IlMjIlM0UlM0N0ciUyMHN0eWxlJTNEJTIyYmFja2dyb3VuZC1jb2xvciUzQSUyMHRyYW5zcGFyZW50JTNCJTIyJTNFJTNDISU1QmVuZGlmJTVELS0lM0UlMEElMjAlMjAlMjAlMjAlMjAlMjAlMEElM0MhLS0lNUJpZiUyMChtc28pJTdDKElFKSU1RCUzRSUzQ3RkJTIwYWxpZ24lM0QlMjJjZW50ZXIlMjIlMjB3aWR0aCUzRCUyMjUwMCUyMiUyMHN0eWxlJTNEJTIyd2lkdGglM0ElMjA1MDBweCUzQnBhZGRpbmclM0ElMjAwcHglM0Jib3JkZXItdG9wJTNBJTIwMHB4JTIwc29saWQlMjB0cmFuc3BhcmVudCUzQmJvcmRlci1sZWZ0JTNBJTIwMHB4JTIwc29saWQlMjB0cmFuc3BhcmVudCUzQmJvcmRlci1yaWdodCUzQSUyMDBweCUyMHNvbGlkJTIwdHJhbnNwYXJlbnQlM0Jib3JkZXItYm90dG9tJTNBJTIwMHB4JTIwc29saWQlMjB0cmFuc3BhcmVudCUzQmJvcmRlci1yYWRpdXMlM0ElMjAwcHglM0Itd2Via2l0LWJvcmRlci1yYWRpdXMlM0ElMjAwcHglM0IlMjAtbW96LWJvcmRlci1yYWRpdXMlM0ElMjAwcHglM0IlMjIlMjB2YWxpZ24lM0QlMjJ0b3AlMjIlM0UlM0MhJTVCZW5kaWYlNUQtLSUzRSUwQSUzQ2RpdiUyMGNsYXNzJTNEJTIydS1jb2wlMjB1LWNvbC0xMDAlMjIlMjBzdHlsZSUzRCUyMm1heC13aWR0aCUzQSUyMDMyMHB4JTNCbWluLXdpZHRoJTNBJTIwNTAwcHglM0JkaXNwbGF5JTNBJTIwdGFibGUtY2VsbCUzQnZlcnRpY2FsLWFsaWduJTNBJTIwdG9wJTNCJTIyJTNFJTBBJTIwJTIwJTNDZGl2JTIwc3R5bGUlM0QlMjJoZWlnaHQlM0ElMjAxMDAlMjUlM0J3aWR0aCUzQSUyMDEwMCUyNSUyMCFpbXBvcnRhbnQlM0Jib3JkZXItcmFkaXVzJTNBJTIwMHB4JTNCLXdlYmtpdC1ib3JkZXItcmFkaXVzJTNBJTIwMHB4JTNCJTIwLW1vei1ib3JkZXItcmFkaXVzJTNBJTIwMHB4JTNCJTIyJTNFJTBBJTIwJTIwJTNDIS0tJTVCaWYlMjAoIW1zbyklMjYoIUlFKSU1RCUzRSUzQyEtLSUzRSUzQ2RpdiUyMHN0eWxlJTNEJTIyYm94LXNpemluZyUzQSUyMGJvcmRlci1ib3glM0IlMjBoZWlnaHQlM0ElMjAxMDAlMjUlM0IlMjBwYWRkaW5nJTNBJTIwMHB4JTNCYm9yZGVyLXRvcCUzQSUyMDBweCUyMHNvbGlkJTIwdHJhbnNwYXJlbnQlM0Jib3JkZXItbGVmdCUzQSUyMDBweCUyMHNvbGlkJTIwdHJhbnNwYXJlbnQlM0Jib3JkZXItcmlnaHQlM0ElMjAwcHglMjBzb2xpZCUyMHRyYW5zcGFyZW50JTNCYm9yZGVyLWJvdHRvbSUzQSUyMDBweCUyMHNvbGlkJTIwdHJhbnNwYXJlbnQlM0Jib3JkZXItcmFkaXVzJTNBJTIwMHB4JTNCLXdlYmtpdC1ib3JkZXItcmFkaXVzJTNBJTIwMHB4JTNCJTIwLW1vei1ib3JkZXItcmFkaXVzJTNBJTIwMHB4JTNCJTIyJTNFJTNDIS0tJTNDISU1QmVuZGlmJTVELS0lM0UlMEElMjAlMjAlMEElM0N0YWJsZSUyMHN0eWxlJTNEJTIyZm9udC1mYW1pbHklM0FhcmlhbCUyQ2hlbHZldGljYSUyQ3NhbnMtc2VyaWYlM0IlMjIlMjByb2xlJTNEJTIycHJlc2VudGF0aW9uJTIyJTIwY2VsbHBhZGRpbmclM0QlMjIwJTIyJTIwY2VsbHNwYWNpbmclM0QlMjIwJTIyJTIwd2lkdGglM0QlMjIxMDAlMjUlMjIlMjBib3JkZXIlM0QlMjIwJTIyJTNFJTBBJTIwJTIwJTNDdGJvZHklM0UlMEElMjAlMjAlMjAlMjAlM0N0ciUzRSUwQSUyMCUyMCUyMCUyMCUyMCUyMCUzQ3RkJTIwc3R5bGUlM0QlMjJvdmVyZmxvdy13cmFwJTNBYnJlYWstd29yZCUzQndvcmQtYnJlYWslM0FicmVhay13b3JkJTNCcGFkZGluZyUzQTEwcHglM0Jmb250LWZhbWlseSUzQWFyaWFsJTJDaGVsdmV0aWNhJTJDc2Fucy1zZXJpZiUzQiUyMiUyMGFsaWduJTNEJTIybGVmdCUyMiUzRSUwQSUyMCUyMCUyMCUyMCUyMCUyMCUyMCUyMCUwQSUyMCUyMCUzQyEtLSU1QmlmJTIwbXNvJTVEJTNFJTNDdGFibGUlMjB3aWR0aCUzRCUyMjEwMCUyNSUyMiUzRSUzQ3RyJTNFJTNDdGQlM0UlM0MhJTVCZW5kaWYlNUQtLSUzRSUwQSUyMCUyMCUyMCUyMCUzQ2gxJTIwc3R5bGUlM0QlMjJtYXJnaW4lM0ElMjAwcHglM0IlMjBsaW5lLWhlaWdodCUzQSUyMDE0MCUyNSUzQiUyMHRleHQtYWxpZ24lM0ElMjBjZW50ZXIlM0IlMjB3b3JkLXdyYXAlM0ElMjBicmVhay13b3JkJTNCJTIwZm9udC1zaXplJTNBJTIwMjJweCUzQiUyMGZvbnQtd2VpZ2h0JTNBJTIwNDAwJTNCJTIyJTNFJTNDc3BhbiUzRVdlbGNvbWUlMjB0byUyME1pcnJvciElM0MlMkZzcGFuJTNFJTNDJTJGaDElM0UlMEElMjAlMjAlM0MhLS0lNUJpZiUyMG1zbyU1RCUzRSUzQyUyRnRkJTNFJTNDJTJGdHIlM0UlM0MlMkZ0YWJsZSUzRSUzQyElNUJlbmRpZiU1RC0tJTNFJTBBJTBBJTIwJTIwJTIwJTIwJTIwJTIwJTNDJTJGdGQlM0UlMEElMjAlMjAlMjAlMjAlM0MlMkZ0ciUzRSUwQSUyMCUyMCUzQyUyRnRib2R5JTNFJTBBJTNDJTJGdGFibGUlM0UlMEElMEElM0N0YWJsZSUyMHN0eWxlJTNEJTIyZm9udC1mYW1pbHklM0FhcmlhbCUyQ2hlbHZldGljYSUyQ3NhbnMtc2VyaWYlM0IlMjIlMjByb2xlJTNEJTIycHJlc2VudGF0aW9uJTIyJTIwY2VsbHBhZGRpbmclM0QlMjIwJTIyJTIwY2VsbHNwYWNpbmclM0QlMjIwJTIyJTIwd2lkdGglM0QlMjIxMDAlMjUlMjIlMjBib3JkZXIlM0QlMjIwJTIyJTNFJTBBJTIwJTIwJTNDdGJvZHklM0UlMEElMjAlMjAlMjAlMjAlM0N0ciUzRSUwQSUyMCUyMCUyMCUyMCUyMCUyMCUzQ3RkJTIwc3R5bGUlM0QlMjJvdmVyZmxvdy13cmFwJTNBYnJlYWstd29yZCUzQndvcmQtYnJlYWslM0FicmVhay13b3JkJTNCcGFkZGluZyUzQTEwcHglM0Jmb250LWZhbWlseSUzQWFyaWFsJTJDaGVsdmV0aWNhJTJDc2Fucy1zZXJpZiUzQiUyMiUyMGFsaWduJTNEJTIybGVmdCUyMiUzRSUwQSUyMCUyMCUyMCUyMCUyMCUyMCUyMCUyMCUwQSUyMCUyMCUzQ3RhYmxlJTIwaGVpZ2h0JTNEJTIyMHB4JTIyJTIwYWxpZ24lM0QlMjJjZW50ZXIlMjIlMjBib3JkZXIlM0QlMjIwJTIyJTIwY2VsbHBhZGRpbmclM0QlMjIwJTIyJTIwY2VsbHNwYWNpbmclM0QlMjIwJTIyJTIwd2lkdGglM0QlMjIxMDAlMjUlMjIlMjBzdHlsZSUzRCUyMmJvcmRlci1jb2xsYXBzZSUzQSUyMGNvbGxhcHNlJTNCdGFibGUtbGF5b3V0JTNBJTIwZml4ZWQlM0Jib3JkZXItc3BhY2luZyUzQSUyMDAlM0Jtc28tdGFibGUtbHNwYWNlJTNBJTIwMHB0JTNCbXNvLXRhYmxlLXJzcGFjZSUzQSUyMDBwdCUzQnZlcnRpY2FsLWFsaWduJTNBJTIwdG9wJTNCYm9yZGVyLXRvcCUzQSUyMDFweCUyMHNvbGlkJTIwJTIzQkJCQkJCJTNCLW1zLXRleHQtc2l6ZS1hZGp1c3QlM0ElMjAxMDAlMjUlM0Itd2Via2l0LXRleHQtc2l6ZS1hZGp1c3QlM0ElMjAxMDAlMjUlMjIlM0UlMEElMjAlMjAlMjAlMjAlM0N0Ym9keSUzRSUwQSUyMCUyMCUyMCUyMCUyMCUyMCUzQ3RyJTIwc3R5bGUlM0QlMjJ2ZXJ0aWNhbC1hbGlnbiUzQSUyMHRvcCUyMiUzRSUwQSUyMCUyMCUyMCUyMCUyMCUyMCUyMCUyMCUzQ3RkJTIwc3R5bGUlM0QlMjJ3b3JkLWJyZWFrJTNBJTIwYnJlYWstd29yZCUzQmJvcmRlci1jb2xsYXBzZSUzQSUyMGNvbGxhcHNlJTIwIWltcG9ydGFudCUzQnZlcnRpY2FsLWFsaWduJTNBJTIwdG9wJTNCZm9udC1zaXplJTNBJTIwMHB4JTNCbGluZS1oZWlnaHQlM0ElMjAwcHglM0Jtc28tbGluZS1oZWlnaHQtcnVsZSUzQSUyMGV4YWN0bHklM0ItbXMtdGV4dC1zaXplLWFkanVzdCUzQSUyMDEwMCUyNSUzQi13ZWJraXQtdGV4dC1zaXplLWFkanVzdCUzQSUyMDEwMCUyNSUyMiUzRSUwQSUyMCUyMCUyMCUyMCUyMCUyMCUyMCUyMCUyMCUyMCUzQ3NwYW4lM0UlMjYlMjMxNjAlM0IlM0MlMkZzcGFuJTNFJTBBJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTNDJTJGdGQlM0UlMEElMjAlMjAlMjAlMjAlMjAlMjAlM0MlMkZ0ciUzRSUwQSUyMCUyMCUyMCUyMCUzQyUyRnRib2R5JTNFJTBBJTIwJTIwJTNDJTJGdGFibGUlM0UlMEElMEElMjAlMjAlMjAlMjAlMjAlMjAlM0MlMkZ0ZCUzRSUwQSUyMCUyMCUyMCUyMCUzQyUyRnRyJTNFJTBBJTIwJTIwJTNDJTJGdGJvZHklM0UlMEElM0MlMkZ0YWJsZSUzRSUwQSUwQSUzQ3RhYmxlJTIwc3R5bGUlM0QlMjJmb250LWZhbWlseSUzQWFyaWFsJTJDaGVsdmV0aWNhJTJDc2Fucy1zZXJpZiUzQiUyMiUyMHJvbGUlM0QlMjJwcmVzZW50YXRpb24lMjIlMjBjZWxscGFkZGluZyUzRCUyMjAlMjIlMjBjZWxsc3BhY2luZyUzRCUyMjAlMjIlMjB3aWR0aCUzRCUyMjEwMCUyNSUyMiUyMGJvcmRlciUzRCUyMjAlMjIlM0UlMEElMjAlMjAlM0N0Ym9keSUzRSUwQSUyMCUyMCUyMCUyMCUzQ3RyJTNFJTBBJTIwJTIwJTIwJTIwJTIwJTIwJTNDdGQlMjBzdHlsZSUzRCUyMm92ZXJmbG93LXdyYXAlM0FicmVhay13b3JkJTNCd29yZC1icmVhayUzQWJyZWFrLXdvcmQlM0JwYWRkaW5nJTNBMTBweCUzQmZvbnQtZmFtaWx5JTNBYXJpYWwlMkNoZWx2ZXRpY2ElMkNzYW5zLXNlcmlmJTNCJTIyJTIwYWxpZ24lM0QlMjJsZWZ0JTIyJTNFJTBBJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTIwJTBBJTIwJTIwJTNDZGl2JTIwc3R5bGUlM0QlMjJmb250LXNpemUlM0ElMjAxNHB4JTNCJTIwbGluZS1oZWlnaHQlM0ElMjAxNDAlMjUlM0IlMjB0ZXh0LWFsaWduJTNBJTIwY2VudGVyJTNCJTIwd29yZC13cmFwJTNBJTIwYnJlYWstd29yZCUzQiUyMiUzRSUwQSUyMCUyMCUyMCUyMCUzQ3AlMjBzdHlsZSUzRCUyMmxpbmUtaGVpZ2h0JTNBJTIwMTQwJTI1JTNCJTIyJTNFQ2xpY2slMjBvbiUyMHRoZSUyMGJ1dHRvbiUyMGJlbG93JTIwdG8lMjB2aXNpdCUyMG91ciUyMHBhZ2UuJTNDJTJGcCUzRSUwQSUyMCUyMCUzQyUyRmRpdiUzRSUwQSUwQSUyMCUyMCUyMCUyMCUyMCUyMCUzQyUyRnRkJTNFJTBBJTIwJTIwJTIwJTIwJTNDJTJGdHIlM0UlMEElMjAlMjAlM0MlMkZ0Ym9keSUzRSUwQSUzQyUyRnRhYmxlJTNFJTBBJTBBJTNDdGFibGUlMjBzdHlsZSUzRCUyMmZvbnQtZmFtaWx5JTNBYXJpYWwlMkNoZWx2ZXRpY2ElMkNzYW5zLXNlcmlmJTNCJTIyJTIwcm9sZSUzRCUyMnByZXNlbnRhdGlvbiUyMiUyMGNlbGxwYWRkaW5nJTNEJTIyMCUyMiUyMGNlbGxzcGFjaW5nJTNEJTIyMCUyMiUyMHdpZHRoJTNEJTIyMTAwJTI1JTIyJTIwYm9yZGVyJTNEJTIyMCUyMiUzRSUwQSUyMCUyMCUzQ3Rib2R5JTNFJTBBJTIwJTIwJTIwJTIwJTNDdHIlM0UlMEElMjAlMjAlMjAlMjAlMjAlMjAlM0N0ZCUyMHN0eWxlJTNEJTIyb3ZlcmZsb3ctd3JhcCUzQWJyZWFrLXdvcmQlM0J3b3JkLWJyZWFrJTNBYnJlYWstd29yZCUzQnBhZGRpbmclM0ExMHB4JTNCZm9udC1mYW1pbHklM0FhcmlhbCUyQ2hlbHZldGljYSUyQ3NhbnMtc2VyaWYlM0IlMjIlMjBhbGlnbiUzRCUyMmxlZnQlMjIlM0UlMEElMjAlMjAlMjAlMjAlMjAlMjAlMjAlMjAlMEElMjAlMjAlM0MhLS0lNUJpZiUyMG1zbyU1RCUzRSUzQ3N0eWxlJTNFLnYtYnV0dG9uJTIwJTdCYmFja2dyb3VuZCUzQSUyMHRyYW5zcGFyZW50JTIwIWltcG9ydGFudCUzQiU3RCUzQyUyRnN0eWxlJTNFJTNDISU1QmVuZGlmJTVELS0lM0UlMEElM0NkaXYlMjBhbGlnbiUzRCUyMmNlbnRlciUyMiUzRSUwQSUyMCUyMCUzQyEtLSU1QmlmJTIwbXNvJTVEJTNFJTNDdiUzQXJvdW5kcmVjdCUyMHhtbG5zJTNBdiUzRCUyMnVybiUzQXNjaGVtYXMtbWljcm9zb2Z0LWNvbSUzQXZtbCUyMiUyMHhtbG5zJTNBdyUzRCUyMnVybiUzQXNjaGVtYXMtbWljcm9zb2Z0LWNvbSUzQW9mZmljZSUzQXdvcmQlMjIlMjBocmVmJTNEJTIyaHR0cHMlM0ElMkYlMkZ3d3cubWlycm9yY2RwLmNvbSUyRiUyMiUyMHN0eWxlJTNEJTIyaGVpZ2h0JTNBMzdweCUzQiUyMHYtdGV4dC1hbmNob3IlM0FtaWRkbGUlM0IlMjB3aWR0aCUzQTc3cHglM0IlMjIlMjBhcmNzaXplJTNEJTIyMTElMjUlMjIlMjAlMjBzdHJva2UlM0QlMjJmJTIyJTIwZmlsbGNvbG9yJTNEJTIyJTIzMDE2ZmVlJTIyJTNFJTNDdyUzQWFuY2hvcmxvY2slMkYlM0UlM0NjZW50ZXIlMjBzdHlsZSUzRCUyMmNvbG9yJTNBJTIzRkZGRkZGJTNCJTIyJTNFJTNDISU1QmVuZGlmJTVELS0lM0UlMEElMjAlMjAlMjAlMjAlM0NhJTIwaHJlZiUzRCUyMmh0dHBzJTNBJTJGJTJGd3d3Lm1pcnJvcmNkcC5jb20lMkYlMjIlMjB0YXJnZXQlM0QlMjJfYmxhbmslMjIlMjBjbGFzcyUzRCUyMnYtYnV0dG9uJTIyJTIwc3R5bGUlM0QlMjJib3gtc2l6aW5nJTNBJTIwYm9yZGVyLWJveCUzQmRpc3BsYXklM0ElMjBpbmxpbmUtYmxvY2slM0J0ZXh0LWRlY29yYXRpb24lM0ElMjBub25lJTNCLXdlYmtpdC10ZXh0LXNpemUtYWRqdXN0JTNBJTIwbm9uZSUzQnRleHQtYWxpZ24lM0ElMjBjZW50ZXIlM0Jjb2xvciUzQSUyMCUyM0ZGRkZGRiUzQiUyMGJhY2tncm91bmQtY29sb3IlM0ElMjAlMjMwMTZmZWUlM0IlMjBib3JkZXItcmFkaXVzJTNBJTIwNHB4JTNCLXdlYmtpdC1ib3JkZXItcmFkaXVzJTNBJTIwNHB4JTNCJTIwLW1vei1ib3JkZXItcmFkaXVzJTNBJTIwNHB4JTNCJTIwd2lkdGglM0FhdXRvJTNCJTIwbWF4LXdpZHRoJTNBMTAwJTI1JTNCJTIwb3ZlcmZsb3ctd3JhcCUzQSUyMGJyZWFrLXdvcmQlM0IlMjB3b3JkLWJyZWFrJTNBJTIwYnJlYWstd29yZCUzQiUyMHdvcmQtd3JhcCUzQWJyZWFrLXdvcmQlM0IlMjBtc28tYm9yZGVyLWFsdCUzQSUyMG5vbmUlM0Jmb250LXNpemUlM0ElMjAxNHB4JTNCJTIyJTNFJTBBJTIwJTIwJTIwJTIwJTIwJTIwJTNDc3BhbiUyMHN0eWxlJTNEJTIyZGlzcGxheSUzQWJsb2NrJTNCcGFkZGluZyUzQTEwcHglMjAyMHB4JTNCbGluZS1oZWlnaHQlM0ExMjAlMjUlM0IlMjIlM0UlM0NzcGFuJTIwc3R5bGUlM0QlMjJsaW5lLWhlaWdodCUzQSUyMDE2LjhweCUzQiUyMiUzRU1pcnJvciUzQyUyRnNwYW4lM0UlM0MlMkZzcGFuJTNFJTBBJTIwJTIwJTIwJTIwJTNDJTJGYSUzRSUwQSUyMCUyMCUyMCUyMCUzQyEtLSU1QmlmJTIwbXNvJTVEJTNFJTNDJTJGY2VudGVyJTNFJTNDJTJGdiUzQXJvdW5kcmVjdCUzRSUzQyElNUJlbmRpZiU1RC0tJTNFJTBBJTNDJTJGZGl2JTNFJTBBJTBBJTIwJTIwJTIwJTIwJTIwJTIwJTNDJTJGdGQlM0UlMEElMjAlMjAlMjAlMjAlM0MlMkZ0ciUzRSUwQSUyMCUyMCUzQyUyRnRib2R5JTNFJTBBJTNDJTJGdGFibGUlM0UlMEElMEElMjAlMjAlM0MhLS0lNUJpZiUyMCghbXNvKSUyNighSUUpJTVEJTNFJTNDIS0tJTNFJTNDJTJGZGl2JTNFJTNDIS0tJTNDISU1QmVuZGlmJTVELS0lM0UlMEElMjAlMjAlM0MlMkZkaXYlM0UlMEElM0MlMkZkaXYlM0UlMEElM0MhLS0lNUJpZiUyMChtc28pJTdDKElFKSU1RCUzRSUzQyUyRnRkJTNFJTNDISU1QmVuZGlmJTVELS0lM0UlMEElMjAlMjAlMjAlMjAlMjAlMjAlM0MhLS0lNUJpZiUyMChtc28pJTdDKElFKSU1RCUzRSUzQyUyRnRyJTNFJTNDJTJGdGFibGUlM0UlM0MlMkZ0ZCUzRSUzQyUyRnRyJTNFJTNDJTJGdGFibGUlM0UlM0MhJTVCZW5kaWYlNUQtLSUzRSUwQSUyMCUyMCUyMCUyMCUzQyUyRmRpdiUzRSUwQSUyMCUyMCUzQyUyRmRpdiUzRSUwQSUyMCUyMCUzQyUyRmRpdiUzRSUwQSUyMCUyMCUwQSUwQSUwQSUyMCUyMCUyMCUyMCUzQyEtLSU1QmlmJTIwKG1zbyklN0MoSUUpJTVEJTNFJTNDJTJGdGQlM0UlM0MlMkZ0ciUzRSUzQyUyRnRhYmxlJTNFJTNDISU1QmVuZGlmJTVELS0lM0UlMEElMjAlMjAlMjAlMjAlM0MlMkZ0ZCUzRSUwQSUyMCUyMCUzQyUyRnRyJTNFJTBBJTIwJTIwJTNDJTJGdGJvZHklM0UlMEElMjAlMjAlM0MlMkZ0YWJsZSUzRSUwQSUyMCUyMCUzQyEtLSU1QmlmJTIwbXNvJTVEJTNFJTNDJTJGZGl2JTNFJTNDISU1QmVuZGlmJTVELS0lM0UlMEElMjAlMjAlM0MhLS0lNUJpZiUyMElFJTVEJTNFJTNDJTJGZGl2JTNFJTNDISU1QmVuZGlmJTVELS0lM0UlMEElM0MlMkZib2R5JTNFJTBBJTBBJTNDJTJGaHRtbCUzRSUwQQ=="
)

var dummyEmailJson = fmt.Sprintf(`{"counters":{"u_column":1,"u_row":1,"u_content_text":3,"u_content_divider":1,"u_content_heading":1,"u_content_button":1},"body":{"id":"TZz18h5bCd","rows":[{"id":"lh5PZRVf3u","cells":[1],"columns":[{"id":"AA48ApALEi","contents":[{"id":"SCG_p4fO7n","type":"heading","values":{"containerPadding":"10px","anchor":"","headingType":"h1","fontSize":"22px","textAlign":"center","lineHeight":"140%","linkStyle":{"inherit":true,"linkColor":"#0000ee","linkHoverColor":"#0000ee","linkUnderline":true,"linkHoverUnderline":true},"displayCondition":null,"_styleGuide":null,"_meta":{"htmlID":"u_content_heading_1","htmlClassNames":"u_content_heading"},"selectable":true,"draggable":true,"duplicatable":true,"deletable":true,"hideable":true,"text":"<span>Welcome to Mirror!</span>","_languages":{}}},{"id":"uiK4pRpetX","type":"divider","values":{"width":"100%","border":{"borderTopWidth":"1px","borderTopStyle":"solid","borderTopColor":"#BBBBBB"},"textAlign":"center","containerPadding":"10px","anchor":"","displayCondition":null,"_styleGuide":null,"_meta":{"htmlID":"u_content_divider_1","htmlClassNames":"u_content_divider"},"selectable":true,"draggable":true,"duplicatable":true,"deletable":true,"hideable":true}},{"id":"81CDqbcmae","type":"text","values":{"containerPadding":"10px","fontSize":"14px","textAlign":"center","lineHeight":"140%","linkStyle":{"inherit":true,"linkColor":"#0000ee","linkHoverColor":"#0000ee","linkUnderline":true,"linkHoverUnderline":true},"displayCondition":null,"_styleGuide":null,"_meta":{"htmlID":"u_content_text_3","htmlClassNames":"u_content_text"},"selectable":true,"draggable":true,"duplicatable":true,"deletable":true,"hideable":true,"text":"<p style=\"line-height: 140%;\">Click on the button below to visit our page.</p>","_languages":{}}},{"id":"9qZJEn2ypl","type":"button","values":{"href":{"name":"web","attrs":{"href":"{{href}}","target":"{{target}}"},"values":{"href":"%s","target":"_blank"}},"buttonColors":{"color":"#FFFFFF","backgroundColor":"#016fee","hoverColor":"#FFFFFF","hoverBackgroundColor":"#3AAEE0"},"size":{"autoWidth":true,"width":"100%"},"fontSize":"14px","lineHeight":"120%","textAlign":"center","padding":"10px 20px","border":{},"borderRadius":"4px","displayCondition":null,"_styleGuide":null,"containerPadding":"10px","anchor":"","_meta":{"htmlID":"u_content_button_1","htmlClassNames":"u_content_button"},"selectable":true,"draggable":true,"duplicatable":true,"deletable":true,"hideable":true,"text":"<span style=\"line-height: 16.8px;\">Mirror</span>","_languages":{},"calculatedWidth":77,"calculatedHeight":37}}],"values":{"backgroundColor":"","padding":"0px","border":{},"borderRadius":"0px","_meta":{"htmlID":"u_column_1","htmlClassNames":"u_column"}}}],"values":{"displayCondition":null,"columns":false,"_styleGuide":null,"backgroundColor":"","columnsBackgroundColor":"","backgroundImage":{"url":"","fullWidth":true,"repeat":"no-repeat","size":"custom","position":"center","customPosition":["50%","50%"]},"padding":"0px","anchor":"","hideDesktop":false,"_meta":{"htmlID":"u_row_1","htmlClassNames":"u_row"},"selectable":true,"draggable":true,"duplicatable":true,"deletable":true,"hideable":true}}],"headers":[],"footers":[],"values":{"_styleGuide":null,"popupPosition":"center","popupWidth":"600px","popupHeight":"auto","borderRadius":"10px","contentAlign":"center","contentVerticalAlign":"center","contentWidth":"500px","fontFamily":{"label":"Arial","value":"arial,helvetica,sans-serif"},"textColor":"#000000","popupBackgroundColor":"#FFFFFF","popupBackgroundImage":{"url":"","fullWidth":true,"repeat":"no-repeat","size":"cover","position":"center"},"popupOverlay_backgroundColor":"rgba(0, 0, 0, 0.1)","popupCloseButton_position":"top-right","popupCloseButton_backgroundColor":"#DDDDDD","popupCloseButton_iconColor":"#000000","popupCloseButton_borderRadius":"0px","popupCloseButton_margin":"0px","popupCloseButton_action":{"name":"close_popup","attrs":{"onClick":"document.querySelector('.u-popup-container').style.display = 'none';"}},"language":{},"backgroundColor":"#F7F8F9","preheaderText":"","linkStyle":{"body":true,"linkColor":"#0000ee","linkHoverColor":"#0000ee","linkUnderline":true,"linkHoverUnderline":true},"backgroundImage":{"url":"","fullWidth":true,"repeat":"no-repeat","size":"custom","position":"center"},"_meta":{"htmlID":"u_body","htmlClassNames":"u_body"}}},"schemaVersion":18}`, targetLink)

type AccountHandler interface {
	CreateTrialAccount(ctx context.Context, req *CreateTrialAccountRequest, res *CreateTrialAccountResponse) error
}

type accountHandler struct {
	cfg             *config.Config
	tenantHandler   TenantHandler
	userHandler     UserHandler
	tagHandler      TagHandler
	segmentHandler  SegmentHandler
	emailHandler    EmailHandler
	campaignRepo    repo.CampaignRepo
	queryRepo       repo.QueryRepo
	taskRepo        repo.TaskRepo
	campaignLogRepo repo.CampaignLogRepo
}

func NewAccountHandler(cfg *config.Config, tenantHandler TenantHandler, userHandler UserHandler,
	tagHandler TagHandler, segmentHandler SegmentHandler, emailHandler EmailHandler, campaignRepo repo.CampaignRepo,
	queryRepo repo.QueryRepo, taskRepo repo.TaskRepo, campaignLogRepo repo.CampaignLogRepo) AccountHandler {
	return &accountHandler{
		cfg:             cfg,
		tenantHandler:   tenantHandler,
		userHandler:     userHandler,
		tagHandler:      tagHandler,
		segmentHandler:  segmentHandler,
		emailHandler:    emailHandler,
		campaignRepo:    campaignRepo,
		queryRepo:       queryRepo,
		taskRepo:        taskRepo,
		campaignLogRepo: campaignLogRepo,
	}
}

type CreateTrialAccountRequest struct {
	Token *string `schema:"token,omitempty"`
}

func (r *CreateTrialAccountRequest) GetToken() string {
	if r != nil && r.Token != nil {
		return *r.Token
	}
	return ""
}

type CreateTrialAccountResponse struct {
	Session *entity.Session `json:"session,omitempty"`
}

var CreateTrialAccountValidator = validator.MustForm(map[string]validator.Validator{
	"token": &validator.String{},
})

func (h *accountHandler) CreateTrialAccount(ctx context.Context, req *CreateTrialAccountRequest, res *CreateTrialAccountResponse) error {
	if err := CreateTrialAccountValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	if req.GetToken() != h.cfg.TrialAccountToken {
		return errutil.ValidationError(errors.New("invalid trial account token"))
	}

	// ========== Create Tenant with First User ==========

	var (
		suffix     = strings.ToLower(goutil.GenerateRandString(15))
		tenantName = fmt.Sprintf("demo-mirror-%s", suffix)

		username = "admin"
		email    = fmt.Sprintf("%s@mirror.com", username)
	)

	password, err := goutil.GenerateSecureRandString(15)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("generate random password failed: %v", err)
		return err
	}

	var (
		createTenantReq = &CreateTenantRequest{
			Name: goutil.String(tenantName),
			Users: []*CreateUserRequest{
				{
					Email:    goutil.String(email),
					Password: goutil.String(password),
				},
			},
		}
		createTenantRes = new(CreateTenantResponse)
	)
	if err := h.tenantHandler.CreateTenant(ctx, createTenantReq, createTenantRes); err != nil {
		log.Ctx(ctx).Error().Msgf("create tenant error: %s", err)
		return err
	}

	// ========== Create Two Tags ==========

	var (
		tenant    = createTenantRes.Tenant
		adminUser = createTenantRes.Users[0]
	)

	contextInfo := ContextInfo{
		Tenant: tenant,
		User:   adminUser,
	}

	var (
		createTagReqs = []*CreateTagRequest{
			{
				ContextInfo: contextInfo,
				Name:        goutil.String("Age"),
				TagDesc:     goutil.String("Age of Users"),
				ValueType:   goutil.Uint32(uint32(entity.TagValueTypeInt)),
			},
			{
				ContextInfo: contextInfo,
				Name:        goutil.String("Country"),
				TagDesc:     goutil.String("Country of Users"),
				ValueType:   goutil.Uint32(uint32(entity.TagValueTypeStr)),
			},
		}
		createTagRess = []*CreateTagResponse{
			new(CreateTagResponse),
			new(CreateTagResponse),
		}
	)
	for i, tagReq := range createTagReqs {
		if err := h.tagHandler.CreateTag(ctx, tagReq, createTagRess[i]); err != nil {
			log.Ctx(ctx).Error().Msgf("create tag error: %s", err)
			return err
		}
	}

	// ========== Create Segment ==========

	var (
		ageTag     = createTagRess[0].Tag
		countryTag = createTagRess[1].Tag
	)

	var (
		createSegmentReq = &CreateSegmentRequest{
			ContextInfo: contextInfo,
			Name:        goutil.String("Millennials or Malaysians"),
			SegmentDesc: goutil.String("Users aged between 18 and 40, or users from Malaysia"),
			Criteria: &entity.Query{
				Queries: []*entity.Query{
					{
						Lookups: []*entity.Lookup{
							{
								TagID: ageTag.ID,
								Op:    entity.LookupOpGt,
								Val:   18,
							},
							{
								TagID: ageTag.ID,
								Op:    entity.LookupOpLt,
								Val:   40,
							},
						},
						Op: entity.QueryOpAnd,
					},
					{
						Lookups: []*entity.Lookup{
							{
								TagID: countryTag.ID,
								Op:    entity.LookupOpEq,
								Val:   "Malaysia",
							},
						},
						Op: entity.QueryOpAnd,
					},
				},
				Op: entity.QueryOpOr,
			},
		}
		createSegmentRes = new(CreateSegmentResponse)
	)
	if err := h.segmentHandler.CreateSegment(ctx, createSegmentReq, createSegmentRes); err != nil {
		log.Ctx(ctx).Error().Msgf("create segment error: %s", err)
		return err
	}

	// ========== Create Dummy Tasks ==========

	var (
		now         = uint64(time.Now().Unix())
		userSize    = uint64(20)
		resourceIDs = []uint64{ageTag.GetID(), countryTag.GetID()}
		task        = &entity.Task{
			TenantID:     tenant.ID,
			ResourceID:   nil,
			Status:       entity.TaskStatusSuccess,
			TaskType:     entity.TaskTypeFileUpload,
			ResourceType: entity.ResourceTypeTag,
			ExtInfo: &entity.TaskExtInfo{
				FileID:      goutil.String(""),
				OriFileName: goutil.String("file.csv"),
				Size:        goutil.Uint64(userSize),
				Progress:    goutil.Uint64(100),
			},
			CreatorID:  adminUser.ID,
			CreateTime: goutil.Uint64(now),
			UpdateTime: goutil.Uint64(now),
		}
	)
	for _, resourceID := range resourceIDs {
		task.ResourceID = goutil.Uint64(resourceID)

		_, err := h.taskRepo.Create(ctx, task)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("create task error: %s", err)
			return err
		}
	}

	// ========== Insert UD Tag Vals ==========

	dummyUserEmails := make([]string, 0)
	for i := 0; i < int(userSize); i++ {
		dummyUserEmails = append(dummyUserEmails, fmt.Sprintf("%s@mirror.com", goutil.GenerateRandString(10)))
	}

	udTagVals := make([]*entity.UdTagVal, 0)
	for _, dummyUserEmail := range dummyUserEmails {
		udTagVals = append(udTagVals, &entity.UdTagVal{
			Ud: &entity.Ud{
				ID:     goutil.String(dummyUserEmail),
				IDType: entity.IDTypeEmail,
			},
			TagVals: []*entity.TagVal{
				{
					TagID:  ageTag.ID,
					TagVal: goutil.Uint64(uint64(rand.Intn(48-18+1) + 18)), // between 18 and 48
				},
				{
					TagID:  countryTag.ID,
					TagVal: goutil.String("Malaysia"),
				},
			},
		})
	}

	if err := h.queryRepo.BatchUpsert(ctx, tenantName, udTagVals, nil); err != nil {
		log.Ctx(ctx).Error().Msgf("batch upsert error: %v", err)
		return err
	}

	// ========== Create Email ==========

	var (
		createEmailReq = &CreateEmailRequest{
			ContextInfo: contextInfo,
			Name:        goutil.String("Sample Email"),
			EmailDesc:   goutil.String("This is a sample email"),
			Html:        goutil.String(dummyEmailHtmlEncoded),
			Json:        goutil.String(dummyEmailJson),
		}
		createEmailRes = new(CreateEmailResponse)
	)
	if err := h.emailHandler.CreateEmail(ctx, createEmailReq, createEmailRes); err != nil {
		log.Ctx(ctx).Error().Msgf("create email error: %s", err)
		return err
	}

	// ========== Create Campaign ==========

	campaign := &entity.Campaign{
		Name:         goutil.String("Sample Campaign"),
		CampaignDesc: goutil.String("This is a sample campaign."),
		SegmentID:    createSegmentRes.Segment.ID,
		SenderID:     goutil.Uint64(1),
		SegmentSize:  goutil.Uint64(userSize),
		Progress:     goutil.Uint64(100),
		Status:       entity.CampaignStatusRunning,
		CampaignEmails: []*entity.CampaignEmail{
			{
				EmailID: createEmailRes.Email.ID,
				Subject: goutil.String("A/B Test Subject #1"),
				Ratio:   goutil.Uint64(50),
				CampaignResult: &entity.CampaignResult{
					TotalUniqueOpenCount: goutil.Uint64(5),
					TotalClickCount:      goutil.Uint64(3),
					AvgOpenTime:          goutil.Uint64(now + 3600),
					ClickCountsByLink: map[string]uint64{
						targetLink: 3,
					},
				},
			},
			{
				EmailID: createEmailRes.Email.ID,
				Subject: goutil.String("A/B Test Subject #2"),
				Ratio:   goutil.Uint64(50),
				CampaignResult: &entity.CampaignResult{
					TotalUniqueOpenCount: goutil.Uint64(10),
					TotalClickCount:      goutil.Uint64(5),
					AvgOpenTime:          goutil.Uint64(now + 7200),
					ClickCountsByLink: map[string]uint64{
						targetLink: 5,
					},
				},
			},
		},
		CreatorID:  adminUser.ID,
		TenantID:   tenant.ID,
		Schedule:   goutil.Uint64(now),
		CreateTime: goutil.Uint64(now),
		UpdateTime: goutil.Uint64(now),
	}

	_, err = h.campaignRepo.Create(ctx, campaign)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("create campaign error: %s", err)
		return err
	}

	// ========== Create Campaign Logs ==========

	campaignLogs := make([]*entity.CampaignLog, 0)
	for i := 0; i < 2; i++ {
		for j := 0; j < i+3; j++ {
			campaignLogs = append(campaignLogs, &entity.CampaignLog{
				CampaignEmailID: goutil.Uint64(uint64(i + 1)),
				Event:           entity.EventUniqueOpened,
				Link:            goutil.String(""),
				EventTime:       goutil.Uint64(now + 3600),
				Email:           goutil.String(dummyUserEmails[j]),
				CreateTime:      goutil.Uint64(now + 3600),
			})
		}

		for j := 0; j < i+2; j++ {
			campaignLogs = append(campaignLogs, &entity.CampaignLog{
				CampaignEmailID: goutil.Uint64(uint64(i + 1)),
				Event:           entity.EventClick,
				Link:            goutil.String(targetLink),
				EventTime:       goutil.Uint64(now + 3600),
				Email:           goutil.String(dummyUserEmails[j]),
				CreateTime:      goutil.Uint64(now + 3600),
			})
		}
	}

	if err := h.campaignLogRepo.CreateMany(ctx, campaignLogs); err != nil {
		log.Ctx(ctx).Error().Msgf("create campaign logs error: %v", err)
		return err
	}

	// ========== Log In ==========

	var (
		logInReq = &LogInRequest{
			TenantName: goutil.String(tenantName),
			Username:   goutil.String(username),
			Password:   goutil.String(password),
		}
		logInRes = new(LogInResponse)
	)
	if err := h.userHandler.LogIn(ctx, logInReq, logInRes); err != nil {
		log.Ctx(ctx).Error().Msgf("login user error: %s", err)
		return err
	}

	res.Session = logInRes.Session

	return nil
}
