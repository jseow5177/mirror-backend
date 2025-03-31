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
	dummyEmailHtmlEncoded = "JTNDJTIxRE9DVFlQRStIVE1MK1BVQkxJQyslMjItJTJGJTJGVzNDJTJGJTJGRFREK1hIVE1MKzEuMCtUcmFuc2l0aW9uYWwrJTJGJTJGRU4lMjIrJTIyaHR0cCUzQSUyRiUyRnd3dy53My5vcmclMkZUUiUyRnhodG1sMSUyRkRURCUyRnhodG1sMS10cmFuc2l0aW9uYWwuZHRkJTIyJTNFJTBBJTNDaHRtbCt4bWxucyUzRCUyMmh0dHAlM0ElMkYlMkZ3d3cudzMub3JnJTJGMTk5OSUyRnhodG1sJTIyK3htbG5zJTNBdiUzRCUyMnVybiUzQXNjaGVtYXMtbWljcm9zb2Z0LWNvbSUzQXZtbCUyMit4bWxucyUzQW8lM0QlMjJ1cm4lM0FzY2hlbWFzLW1pY3Jvc29mdC1jb20lM0FvZmZpY2UlM0FvZmZpY2UlMjIlM0UlMEElM0NoZWFkJTNFJTBBJTNDJTIxLS0lNUJpZitndGUrbXNvKzklNUQlM0UlMEElM0N4bWwlM0UlMEErKyUzQ28lM0FPZmZpY2VEb2N1bWVudFNldHRpbmdzJTNFJTBBKysrKyUzQ28lM0FBbGxvd1BORyUyRiUzRSUwQSsrKyslM0NvJTNBUGl4ZWxzUGVySW5jaCUzRTk2JTNDJTJGbyUzQVBpeGVsc1BlckluY2glM0UlMEErKyUzQyUyRm8lM0FPZmZpY2VEb2N1bWVudFNldHRpbmdzJTNFJTBBJTNDJTJGeG1sJTNFJTBBJTNDJTIxJTVCZW5kaWYlNUQtLSUzRSUwQSsrJTNDbWV0YStodHRwLWVxdWl2JTNEJTIyQ29udGVudC1UeXBlJTIyK2NvbnRlbnQlM0QlMjJ0ZXh0JTJGaHRtbCUzQitjaGFyc2V0JTNEVVRGLTglMjIlM0UlMEErKyUzQ21ldGErbmFtZSUzRCUyMnZpZXdwb3J0JTIyK2NvbnRlbnQlM0QlMjJ3aWR0aCUzRGRldmljZS13aWR0aCUyQytpbml0aWFsLXNjYWxlJTNEMS4wJTIyJTNFJTBBKyslM0NtZXRhK25hbWUlM0QlMjJ4LWFwcGxlLWRpc2FibGUtbWVzc2FnZS1yZWZvcm1hdHRpbmclMjIlM0UlMEErKyUzQyUyMS0tJTVCaWYrJTIxbXNvJTVEJTNFJTNDJTIxLS0lM0UlM0NtZXRhK2h0dHAtZXF1aXYlM0QlMjJYLVVBLUNvbXBhdGlibGUlMjIrY29udGVudCUzRCUyMklFJTNEZWRnZSUyMiUzRSUzQyUyMS0tJTNDJTIxJTVCZW5kaWYlNUQtLSUzRSUwQSsrJTNDdGl0bGUlM0UlM0MlMkZ0aXRsZSUzRSUwQSsrJTBBKysrKyUzQ3N0eWxlK3R5cGUlM0QlMjJ0ZXh0JTJGY3NzJTIyJTNFJTBBKysrKysrJTBBKysrKysrJTQwbWVkaWErb25seStzY3JlZW4rYW5kKyUyOG1pbi13aWR0aCUzQSs1MjBweCUyOSslN0IlMEErKysrKysrKy51LXJvdyslN0IlMEErKysrKysrKysrd2lkdGglM0ErNTAwcHgrJTIxaW1wb3J0YW50JTNCJTBBKysrKysrKyslN0QlMEElMEErKysrKysrKy51LXJvdysudS1jb2wrJTdCJTBBKysrKysrKysrK3ZlcnRpY2FsLWFsaWduJTNBK3RvcCUzQiUwQSsrKysrKysrJTdEJTBBJTBBKysrKysrKyslMEErKysrKysrKysrKysudS1yb3crLnUtY29sLTEwMCslN0IlMEErKysrKysrKysrKysrK3dpZHRoJTNBKzUwMHB4KyUyMWltcG9ydGFudCUzQiUwQSsrKysrKysrKysrKyU3RCUwQSsrKysrKysrKyslMEErKysrKyslN0QlMEElMEErKysrKyslNDBtZWRpYStvbmx5K3NjcmVlbithbmQrJTI4bWF4LXdpZHRoJTNBKzUyMHB4JTI5KyU3QiUwQSsrKysrKysrLnUtcm93LWNvbnRhaW5lcislN0IlMEErKysrKysrKysrbWF4LXdpZHRoJTNBKzEwMCUyNSslMjFpbXBvcnRhbnQlM0IlMEErKysrKysrKysrcGFkZGluZy1sZWZ0JTNBKzBweCslMjFpbXBvcnRhbnQlM0IlMEErKysrKysrKysrcGFkZGluZy1yaWdodCUzQSswcHgrJTIxaW1wb3J0YW50JTNCJTBBKysrKysrKyslN0QlMEElMEErKysrKysrKy51LXJvdyslN0IlMEErKysrKysrKysrd2lkdGglM0ErMTAwJTI1KyUyMWltcG9ydGFudCUzQiUwQSsrKysrKysrJTdEJTBBJTBBKysrKysrKysudS1yb3crLnUtY29sKyU3QiUwQSsrKysrKysrKytkaXNwbGF5JTNBK2Jsb2NrKyUyMWltcG9ydGFudCUzQiUwQSsrKysrKysrKyt3aWR0aCUzQSsxMDAlMjUrJTIxaW1wb3J0YW50JTNCJTBBKysrKysrKysrK21pbi13aWR0aCUzQSszMjBweCslMjFpbXBvcnRhbnQlM0IlMEErKysrKysrKysrbWF4LXdpZHRoJTNBKzEwMCUyNSslMjFpbXBvcnRhbnQlM0IlMEErKysrKysrKyU3RCUwQSUwQSsrKysrKysrLnUtcm93Ky51LWNvbCslM0UrZGl2KyU3QiUwQSsrKysrKysrKyttYXJnaW4lM0ErMCthdXRvJTNCJTBBKysrKysrKyslN0QlMEElMEElMEElN0QlMEErKysrJTBBYm9keSU3Qm1hcmdpbiUzQTAlM0JwYWRkaW5nJTNBMCU3RHRhYmxlJTJDdGQlMkN0ciU3QmJvcmRlci1jb2xsYXBzZSUzQWNvbGxhcHNlJTNCdmVydGljYWwtYWxpZ24lM0F0b3AlN0RwJTdCbWFyZ2luJTNBMCU3RC5pZS1jb250YWluZXIrdGFibGUlMkMubXNvLWNvbnRhaW5lcit0YWJsZSU3QnRhYmxlLWxheW91dCUzQWZpeGVkJTdEJTJBJTdCbGluZS1oZWlnaHQlM0Fpbmhlcml0JTdEYSU1QngtYXBwbGUtZGF0YS1kZXRlY3RvcnMlM0R0cnVlJTVEJTdCY29sb3IlM0Fpbmhlcml0JTIxaW1wb3J0YW50JTNCdGV4dC1kZWNvcmF0aW9uJTNBbm9uZSUyMWltcG9ydGFudCU3RCUwQSUwQSUwQXRhYmxlJTJDK3RkKyU3Qitjb2xvciUzQSslMjMwMDAwMDAlM0IrJTdEKyUyM3VfYm9keSthKyU3Qitjb2xvciUzQSslMjMwMDAwZWUlM0IrdGV4dC1kZWNvcmF0aW9uJTNBK3VuZGVybGluZSUzQislN0QlMEErKysrJTNDJTJGc3R5bGUlM0UlMEErKyUwQSsrJTBBJTBBJTNDJTJGaGVhZCUzRSUwQSUwQSUzQ2JvZHkrY2xhc3MlM0QlMjJjbGVhbi1ib2R5K3VfYm9keSUyMitzdHlsZSUzRCUyMm1hcmdpbiUzQSswJTNCcGFkZGluZyUzQSswJTNCLXdlYmtpdC10ZXh0LXNpemUtYWRqdXN0JTNBKzEwMCUyNSUzQmJhY2tncm91bmQtY29sb3IlM0ErJTIzRjdGOEY5JTNCY29sb3IlM0ErJTIzMDAwMDAwJTIyJTNFJTBBKyslM0MlMjEtLSU1QmlmK0lFJTVEJTNFJTNDZGl2K2NsYXNzJTNEJTIyaWUtY29udGFpbmVyJTIyJTNFJTNDJTIxJTVCZW5kaWYlNUQtLSUzRSUwQSsrJTNDJTIxLS0lNUJpZittc28lNUQlM0UlM0NkaXYrY2xhc3MlM0QlMjJtc28tY29udGFpbmVyJTIyJTNFJTNDJTIxJTVCZW5kaWYlNUQtLSUzRSUwQSsrJTNDdGFibGUraWQlM0QlMjJ1X2JvZHklMjIrc3R5bGUlM0QlMjJib3JkZXItY29sbGFwc2UlM0ErY29sbGFwc2UlM0J0YWJsZS1sYXlvdXQlM0ErZml4ZWQlM0Jib3JkZXItc3BhY2luZyUzQSswJTNCbXNvLXRhYmxlLWxzcGFjZSUzQSswcHQlM0Jtc28tdGFibGUtcnNwYWNlJTNBKzBwdCUzQnZlcnRpY2FsLWFsaWduJTNBK3RvcCUzQm1pbi13aWR0aCUzQSszMjBweCUzQk1hcmdpbiUzQSswK2F1dG8lM0JiYWNrZ3JvdW5kLWNvbG9yJTNBKyUyM0Y3RjhGOSUzQndpZHRoJTNBMTAwJTI1JTIyK2NlbGxwYWRkaW5nJTNEJTIyMCUyMitjZWxsc3BhY2luZyUzRCUyMjAlMjIlM0UlMEErKyUzQ3Rib2R5JTNFJTBBKyslM0N0citzdHlsZSUzRCUyMnZlcnRpY2FsLWFsaWduJTNBK3RvcCUyMiUzRSUwQSsrKyslM0N0ZCtzdHlsZSUzRCUyMndvcmQtYnJlYWslM0ErYnJlYWstd29yZCUzQmJvcmRlci1jb2xsYXBzZSUzQStjb2xsYXBzZSslMjFpbXBvcnRhbnQlM0J2ZXJ0aWNhbC1hbGlnbiUzQSt0b3AlMjIlM0UlMEErKysrJTNDJTIxLS0lNUJpZislMjhtc28lMjklN0MlMjhJRSUyOSU1RCUzRSUzQ3RhYmxlK3dpZHRoJTNEJTIyMTAwJTI1JTIyK2NlbGxwYWRkaW5nJTNEJTIyMCUyMitjZWxsc3BhY2luZyUzRCUyMjAlMjIrYm9yZGVyJTNEJTIyMCUyMiUzRSUzQ3RyJTNFJTNDdGQrYWxpZ24lM0QlMjJjZW50ZXIlMjIrc3R5bGUlM0QlMjJiYWNrZ3JvdW5kLWNvbG9yJTNBKyUyM0Y3RjhGOSUzQiUyMiUzRSUzQyUyMSU1QmVuZGlmJTVELS0lM0UlMEErKysrJTBBKyslMEErKyUwQSUzQ2RpditjbGFzcyUzRCUyMnUtcm93LWNvbnRhaW5lciUyMitzdHlsZSUzRCUyMnBhZGRpbmclM0ErMHB4JTNCYmFja2dyb3VuZC1jb2xvciUzQSt0cmFuc3BhcmVudCUyMiUzRSUwQSsrJTNDZGl2K2NsYXNzJTNEJTIydS1yb3clMjIrc3R5bGUlM0QlMjJtYXJnaW4lM0ErMCthdXRvJTNCbWluLXdpZHRoJTNBKzMyMHB4JTNCbWF4LXdpZHRoJTNBKzUwMHB4JTNCb3ZlcmZsb3ctd3JhcCUzQSticmVhay13b3JkJTNCd29yZC13cmFwJTNBK2JyZWFrLXdvcmQlM0J3b3JkLWJyZWFrJTNBK2JyZWFrLXdvcmQlM0JiYWNrZ3JvdW5kLWNvbG9yJTNBK3RyYW5zcGFyZW50JTNCJTIyJTNFJTBBKysrKyUzQ2RpditzdHlsZSUzRCUyMmJvcmRlci1jb2xsYXBzZSUzQStjb2xsYXBzZSUzQmRpc3BsYXklM0ErdGFibGUlM0J3aWR0aCUzQSsxMDAlMjUlM0JoZWlnaHQlM0ErMTAwJTI1JTNCYmFja2dyb3VuZC1jb2xvciUzQSt0cmFuc3BhcmVudCUzQiUyMiUzRSUwQSsrKysrKyUzQyUyMS0tJTVCaWYrJTI4bXNvJTI5JTdDJTI4SUUlMjklNUQlM0UlM0N0YWJsZSt3aWR0aCUzRCUyMjEwMCUyNSUyMitjZWxscGFkZGluZyUzRCUyMjAlMjIrY2VsbHNwYWNpbmclM0QlMjIwJTIyK2JvcmRlciUzRCUyMjAlMjIlM0UlM0N0ciUzRSUzQ3RkK3N0eWxlJTNEJTIycGFkZGluZyUzQSswcHglM0JiYWNrZ3JvdW5kLWNvbG9yJTNBK3RyYW5zcGFyZW50JTNCJTIyK2FsaWduJTNEJTIyY2VudGVyJTIyJTNFJTNDdGFibGUrY2VsbHBhZGRpbmclM0QlMjIwJTIyK2NlbGxzcGFjaW5nJTNEJTIyMCUyMitib3JkZXIlM0QlMjIwJTIyK3N0eWxlJTNEJTIyd2lkdGglM0E1MDBweCUzQiUyMiUzRSUzQ3RyK3N0eWxlJTNEJTIyYmFja2dyb3VuZC1jb2xvciUzQSt0cmFuc3BhcmVudCUzQiUyMiUzRSUzQyUyMSU1QmVuZGlmJTVELS0lM0UlMEErKysrKyslMEElM0MlMjEtLSU1QmlmKyUyOG1zbyUyOSU3QyUyOElFJTI5JTVEJTNFJTNDdGQrYWxpZ24lM0QlMjJjZW50ZXIlMjIrd2lkdGglM0QlMjI1MDAlMjIrc3R5bGUlM0QlMjJ3aWR0aCUzQSs1MDBweCUzQnBhZGRpbmclM0ErMHB4JTNCYm9yZGVyLXRvcCUzQSswcHgrc29saWQrdHJhbnNwYXJlbnQlM0Jib3JkZXItbGVmdCUzQSswcHgrc29saWQrdHJhbnNwYXJlbnQlM0Jib3JkZXItcmlnaHQlM0ErMHB4K3NvbGlkK3RyYW5zcGFyZW50JTNCYm9yZGVyLWJvdHRvbSUzQSswcHgrc29saWQrdHJhbnNwYXJlbnQlM0Jib3JkZXItcmFkaXVzJTNBKzBweCUzQi13ZWJraXQtYm9yZGVyLXJhZGl1cyUzQSswcHglM0IrLW1vei1ib3JkZXItcmFkaXVzJTNBKzBweCUzQiUyMit2YWxpZ24lM0QlMjJ0b3AlMjIlM0UlM0MlMjElNUJlbmRpZiU1RC0tJTNFJTBBJTNDZGl2K2NsYXNzJTNEJTIydS1jb2wrdS1jb2wtMTAwJTIyK3N0eWxlJTNEJTIybWF4LXdpZHRoJTNBKzMyMHB4JTNCbWluLXdpZHRoJTNBKzUwMHB4JTNCZGlzcGxheSUzQSt0YWJsZS1jZWxsJTNCdmVydGljYWwtYWxpZ24lM0ErdG9wJTNCJTIyJTNFJTBBKyslM0NkaXYrc3R5bGUlM0QlMjJoZWlnaHQlM0ErMTAwJTI1JTNCd2lkdGglM0ErMTAwJTI1KyUyMWltcG9ydGFudCUzQmJvcmRlci1yYWRpdXMlM0ErMHB4JTNCLXdlYmtpdC1ib3JkZXItcmFkaXVzJTNBKzBweCUzQistbW96LWJvcmRlci1yYWRpdXMlM0ErMHB4JTNCJTIyJTNFJTBBKyslM0MlMjEtLSU1QmlmKyUyOCUyMW1zbyUyOSUyNiUyOCUyMUlFJTI5JTVEJTNFJTNDJTIxLS0lM0UlM0NkaXYrc3R5bGUlM0QlMjJib3gtc2l6aW5nJTNBK2JvcmRlci1ib3glM0IraGVpZ2h0JTNBKzEwMCUyNSUzQitwYWRkaW5nJTNBKzBweCUzQmJvcmRlci10b3AlM0ErMHB4K3NvbGlkK3RyYW5zcGFyZW50JTNCYm9yZGVyLWxlZnQlM0ErMHB4K3NvbGlkK3RyYW5zcGFyZW50JTNCYm9yZGVyLXJpZ2h0JTNBKzBweCtzb2xpZCt0cmFuc3BhcmVudCUzQmJvcmRlci1ib3R0b20lM0ErMHB4K3NvbGlkK3RyYW5zcGFyZW50JTNCYm9yZGVyLXJhZGl1cyUzQSswcHglM0Itd2Via2l0LWJvcmRlci1yYWRpdXMlM0ErMHB4JTNCKy1tb3otYm9yZGVyLXJhZGl1cyUzQSswcHglM0IlMjIlM0UlM0MlMjEtLSUzQyUyMSU1QmVuZGlmJTVELS0lM0UlMEErKyUwQSUzQ3RhYmxlK3N0eWxlJTNEJTIyZm9udC1mYW1pbHklM0FhcmlhbCUyQ2hlbHZldGljYSUyQ3NhbnMtc2VyaWYlM0IlMjIrcm9sZSUzRCUyMnByZXNlbnRhdGlvbiUyMitjZWxscGFkZGluZyUzRCUyMjAlMjIrY2VsbHNwYWNpbmclM0QlMjIwJTIyK3dpZHRoJTNEJTIyMTAwJTI1JTIyK2JvcmRlciUzRCUyMjAlMjIlM0UlMEErKyUzQ3Rib2R5JTNFJTBBKysrKyUzQ3RyJTNFJTBBKysrKysrJTNDdGQrc3R5bGUlM0QlMjJvdmVyZmxvdy13cmFwJTNBYnJlYWstd29yZCUzQndvcmQtYnJlYWslM0FicmVhay13b3JkJTNCcGFkZGluZyUzQTEwcHglM0Jmb250LWZhbWlseSUzQWFyaWFsJTJDaGVsdmV0aWNhJTJDc2Fucy1zZXJpZiUzQiUyMithbGlnbiUzRCUyMmxlZnQlMjIlM0UlMEErKysrKysrKyUwQSsrJTNDJTIxLS0lNUJpZittc28lNUQlM0UlM0N0YWJsZSt3aWR0aCUzRCUyMjEwMCUyNSUyMiUzRSUzQ3RyJTNFJTNDdGQlM0UlM0MlMjElNUJlbmRpZiU1RC0tJTNFJTBBKysrKyUzQ2gxK3N0eWxlJTNEJTIybWFyZ2luJTNBKzBweCUzQitsaW5lLWhlaWdodCUzQSsxNDAlMjUlM0IrdGV4dC1hbGlnbiUzQStjZW50ZXIlM0Ird29yZC13cmFwJTNBK2JyZWFrLXdvcmQlM0IrZm9udC1zaXplJTNBKzIycHglM0IrZm9udC13ZWlnaHQlM0ErNDAwJTNCJTIyJTNFJTNDc3BhbiUzRVdlbGNvbWUrdG8rTWlycm9yJTIxJTNDJTJGc3BhbiUzRSUzQyUyRmgxJTNFJTBBKyslM0MlMjEtLSU1QmlmK21zbyU1RCUzRSUzQyUyRnRkJTNFJTNDJTJGdHIlM0UlM0MlMkZ0YWJsZSUzRSUzQyUyMSU1QmVuZGlmJTVELS0lM0UlMEElMEErKysrKyslM0MlMkZ0ZCUzRSUwQSsrKyslM0MlMkZ0ciUzRSUwQSsrJTNDJTJGdGJvZHklM0UlMEElM0MlMkZ0YWJsZSUzRSUwQSUwQSUzQ3RhYmxlK3N0eWxlJTNEJTIyZm9udC1mYW1pbHklM0FhcmlhbCUyQ2hlbHZldGljYSUyQ3NhbnMtc2VyaWYlM0IlMjIrcm9sZSUzRCUyMnByZXNlbnRhdGlvbiUyMitjZWxscGFkZGluZyUzRCUyMjAlMjIrY2VsbHNwYWNpbmclM0QlMjIwJTIyK3dpZHRoJTNEJTIyMTAwJTI1JTIyK2JvcmRlciUzRCUyMjAlMjIlM0UlMEErKyUzQ3Rib2R5JTNFJTBBKysrKyUzQ3RyJTNFJTBBKysrKysrJTNDdGQrc3R5bGUlM0QlMjJvdmVyZmxvdy13cmFwJTNBYnJlYWstd29yZCUzQndvcmQtYnJlYWslM0FicmVhay13b3JkJTNCcGFkZGluZyUzQTEwcHglM0Jmb250LWZhbWlseSUzQWFyaWFsJTJDaGVsdmV0aWNhJTJDc2Fucy1zZXJpZiUzQiUyMithbGlnbiUzRCUyMmxlZnQlMjIlM0UlMEErKysrKysrKyUwQSsrJTNDdGFibGUraGVpZ2h0JTNEJTIyMHB4JTIyK2FsaWduJTNEJTIyY2VudGVyJTIyK2JvcmRlciUzRCUyMjAlMjIrY2VsbHBhZGRpbmclM0QlMjIwJTIyK2NlbGxzcGFjaW5nJTNEJTIyMCUyMit3aWR0aCUzRCUyMjEwMCUyNSUyMitzdHlsZSUzRCUyMmJvcmRlci1jb2xsYXBzZSUzQStjb2xsYXBzZSUzQnRhYmxlLWxheW91dCUzQStmaXhlZCUzQmJvcmRlci1zcGFjaW5nJTNBKzAlM0Jtc28tdGFibGUtbHNwYWNlJTNBKzBwdCUzQm1zby10YWJsZS1yc3BhY2UlM0ErMHB0JTNCdmVydGljYWwtYWxpZ24lM0ErdG9wJTNCYm9yZGVyLXRvcCUzQSsxcHgrc29saWQrJTIzQkJCQkJCJTNCLW1zLXRleHQtc2l6ZS1hZGp1c3QlM0ErMTAwJTI1JTNCLXdlYmtpdC10ZXh0LXNpemUtYWRqdXN0JTNBKzEwMCUyNSUyMiUzRSUwQSsrKyslM0N0Ym9keSUzRSUwQSsrKysrKyUzQ3RyK3N0eWxlJTNEJTIydmVydGljYWwtYWxpZ24lM0ErdG9wJTIyJTNFJTBBKysrKysrKyslM0N0ZCtzdHlsZSUzRCUyMndvcmQtYnJlYWslM0ErYnJlYWstd29yZCUzQmJvcmRlci1jb2xsYXBzZSUzQStjb2xsYXBzZSslMjFpbXBvcnRhbnQlM0J2ZXJ0aWNhbC1hbGlnbiUzQSt0b3AlM0Jmb250LXNpemUlM0ErMHB4JTNCbGluZS1oZWlnaHQlM0ErMHB4JTNCbXNvLWxpbmUtaGVpZ2h0LXJ1bGUlM0ErZXhhY3RseSUzQi1tcy10ZXh0LXNpemUtYWRqdXN0JTNBKzEwMCUyNSUzQi13ZWJraXQtdGV4dC1zaXplLWFkanVzdCUzQSsxMDAlMjUlMjIlM0UlMEErKysrKysrKysrJTNDc3BhbiUzRSUyNiUyMzE2MCUzQiUzQyUyRnNwYW4lM0UlMEErKysrKysrKyUzQyUyRnRkJTNFJTBBKysrKysrJTNDJTJGdHIlM0UlMEErKysrJTNDJTJGdGJvZHklM0UlMEErKyUzQyUyRnRhYmxlJTNFJTBBJTBBKysrKysrJTNDJTJGdGQlM0UlMEErKysrJTNDJTJGdHIlM0UlMEErKyUzQyUyRnRib2R5JTNFJTBBJTNDJTJGdGFibGUlM0UlMEElMEElM0N0YWJsZStzdHlsZSUzRCUyMmZvbnQtZmFtaWx5JTNBYXJpYWwlMkNoZWx2ZXRpY2ElMkNzYW5zLXNlcmlmJTNCJTIyK3JvbGUlM0QlMjJwcmVzZW50YXRpb24lMjIrY2VsbHBhZGRpbmclM0QlMjIwJTIyK2NlbGxzcGFjaW5nJTNEJTIyMCUyMit3aWR0aCUzRCUyMjEwMCUyNSUyMitib3JkZXIlM0QlMjIwJTIyJTNFJTBBKyslM0N0Ym9keSUzRSUwQSsrKyslM0N0ciUzRSUwQSsrKysrKyUzQ3RkK3N0eWxlJTNEJTIyb3ZlcmZsb3ctd3JhcCUzQWJyZWFrLXdvcmQlM0J3b3JkLWJyZWFrJTNBYnJlYWstd29yZCUzQnBhZGRpbmclM0ExMHB4JTNCZm9udC1mYW1pbHklM0FhcmlhbCUyQ2hlbHZldGljYSUyQ3NhbnMtc2VyaWYlM0IlMjIrYWxpZ24lM0QlMjJsZWZ0JTIyJTNFJTBBKysrKysrKyslMEErKyUzQ2RpditzdHlsZSUzRCUyMmZvbnQtc2l6ZSUzQSsxNHB4JTNCK2xpbmUtaGVpZ2h0JTNBKzE0MCUyNSUzQit0ZXh0LWFsaWduJTNBK2NlbnRlciUzQit3b3JkLXdyYXAlM0ErYnJlYWstd29yZCUzQiUyMiUzRSUwQSsrKyslM0NwK3N0eWxlJTNEJTIybGluZS1oZWlnaHQlM0ErMTQwJTI1JTNCJTIyJTNFQ2xpY2srb24rdGhlK2J1dHRvbitiZWxvdyt0byt2aXNpdCtvdXIrcGFnZS4lM0MlMkZwJTNFJTBBKyslM0MlMkZkaXYlM0UlMEElMEErKysrKyslM0MlMkZ0ZCUzRSUwQSsrKyslM0MlMkZ0ciUzRSUwQSsrJTNDJTJGdGJvZHklM0UlMEElM0MlMkZ0YWJsZSUzRSUwQSUwQSUzQ3RhYmxlK3N0eWxlJTNEJTIyZm9udC1mYW1pbHklM0FhcmlhbCUyQ2hlbHZldGljYSUyQ3NhbnMtc2VyaWYlM0IlMjIrcm9sZSUzRCUyMnByZXNlbnRhdGlvbiUyMitjZWxscGFkZGluZyUzRCUyMjAlMjIrY2VsbHNwYWNpbmclM0QlMjIwJTIyK3dpZHRoJTNEJTIyMTAwJTI1JTIyK2JvcmRlciUzRCUyMjAlMjIlM0UlMEErKyUzQ3Rib2R5JTNFJTBBKysrKyUzQ3RyJTNFJTBBKysrKysrJTNDdGQrc3R5bGUlM0QlMjJvdmVyZmxvdy13cmFwJTNBYnJlYWstd29yZCUzQndvcmQtYnJlYWslM0FicmVhay13b3JkJTNCcGFkZGluZyUzQTEwcHglM0Jmb250LWZhbWlseSUzQWFyaWFsJTJDaGVsdmV0aWNhJTJDc2Fucy1zZXJpZiUzQiUyMithbGlnbiUzRCUyMmxlZnQlMjIlM0UlMEErKysrKysrKyUwQSsrJTNDJTIxLS0lNUJpZittc28lNUQlM0UlM0NzdHlsZSUzRS52LWJ1dHRvbislN0JiYWNrZ3JvdW5kJTNBK3RyYW5zcGFyZW50KyUyMWltcG9ydGFudCUzQiU3RCUzQyUyRnN0eWxlJTNFJTNDJTIxJTVCZW5kaWYlNUQtLSUzRSUwQSUzQ2RpdithbGlnbiUzRCUyMmNlbnRlciUyMiUzRSUwQSsrJTNDJTIxLS0lNUJpZittc28lNUQlM0UlM0N2JTNBcm91bmRyZWN0K3htbG5zJTNBdiUzRCUyMnVybiUzQXNjaGVtYXMtbWljcm9zb2Z0LWNvbSUzQXZtbCUyMit4bWxucyUzQXclM0QlMjJ1cm4lM0FzY2hlbWFzLW1pY3Jvc29mdC1jb20lM0FvZmZpY2UlM0F3b3JkJTIyK2hyZWYlM0QlMjJodHRwcyUzQSUyRiUyRnd3dy5taXJyb3JjZHAuY29tJTJGJTIyK3N0eWxlJTNEJTIyaGVpZ2h0JTNBMzdweCUzQit2LXRleHQtYW5jaG9yJTNBbWlkZGxlJTNCK3dpZHRoJTNBNzdweCUzQiUyMithcmNzaXplJTNEJTIyMTElMjUlMjIrK3N0cm9rZSUzRCUyMmYlMjIrZmlsbGNvbG9yJTNEJTIyJTIzMDE2ZmVlJTIyJTNFJTNDdyUzQWFuY2hvcmxvY2slMkYlM0UlM0NjZW50ZXIrc3R5bGUlM0QlMjJjb2xvciUzQSUyM0ZGRkZGRiUzQiUyMiUzRSUzQyUyMSU1QmVuZGlmJTVELS0lM0UlMEErKysrJTNDYStocmVmJTNEJTIyaHR0cHMlM0ElMkYlMkZ3d3cubWlycm9yY2RwLmNvbSUyRiUyMit0YXJnZXQlM0QlMjJfYmxhbmslMjIrY2xhc3MlM0QlMjJ2LWJ1dHRvbiUyMitzdHlsZSUzRCUyMmJveC1zaXppbmclM0ErYm9yZGVyLWJveCUzQmRpc3BsYXklM0EraW5saW5lLWJsb2NrJTNCdGV4dC1kZWNvcmF0aW9uJTNBK25vbmUlM0Itd2Via2l0LXRleHQtc2l6ZS1hZGp1c3QlM0Erbm9uZSUzQnRleHQtYWxpZ24lM0ErY2VudGVyJTNCY29sb3IlM0ErJTIzRkZGRkZGJTNCK2JhY2tncm91bmQtY29sb3IlM0ErJTIzMDE2ZmVlJTNCK2JvcmRlci1yYWRpdXMlM0ErNHB4JTNCLXdlYmtpdC1ib3JkZXItcmFkaXVzJTNBKzRweCUzQistbW96LWJvcmRlci1yYWRpdXMlM0ErNHB4JTNCK3dpZHRoJTNBYXV0byUzQittYXgtd2lkdGglM0ExMDAlMjUlM0Irb3ZlcmZsb3ctd3JhcCUzQSticmVhay13b3JkJTNCK3dvcmQtYnJlYWslM0ErYnJlYWstd29yZCUzQit3b3JkLXdyYXAlM0FicmVhay13b3JkJTNCK21zby1ib3JkZXItYWx0JTNBK25vbmUlM0Jmb250LXNpemUlM0ErMTRweCUzQiUyMiUzRSUwQSsrKysrKyUzQ3NwYW4rc3R5bGUlM0QlMjJkaXNwbGF5JTNBYmxvY2slM0JwYWRkaW5nJTNBMTBweCsyMHB4JTNCbGluZS1oZWlnaHQlM0ExMjAlMjUlM0IlMjIlM0UlM0NzcGFuK3N0eWxlJTNEJTIybGluZS1oZWlnaHQlM0ErMTYuOHB4JTNCJTIyJTNFTWlycm9yJTNDJTJGc3BhbiUzRSUzQyUyRnNwYW4lM0UlMEErKysrJTNDJTJGYSUzRSUwQSsrKyslM0MlMjEtLSU1QmlmK21zbyU1RCUzRSUzQyUyRmNlbnRlciUzRSUzQyUyRnYlM0Fyb3VuZHJlY3QlM0UlM0MlMjElNUJlbmRpZiU1RC0tJTNFJTBBJTNDJTJGZGl2JTNFJTBBJTBBKysrKysrJTNDJTJGdGQlM0UlMEErKysrJTNDJTJGdHIlM0UlMEErKyUzQyUyRnRib2R5JTNFJTBBJTNDJTJGdGFibGUlM0UlMEElMEErKyUzQyUyMS0tJTVCaWYrJTI4JTIxbXNvJTI5JTI2JTI4JTIxSUUlMjklNUQlM0UlM0MlMjEtLSUzRSUzQyUyRmRpdiUzRSUzQyUyMS0tJTNDJTIxJTVCZW5kaWYlNUQtLSUzRSUwQSsrJTNDJTJGZGl2JTNFJTBBJTNDJTJGZGl2JTNFJTBBJTNDJTIxLS0lNUJpZislMjhtc28lMjklN0MlMjhJRSUyOSU1RCUzRSUzQyUyRnRkJTNFJTNDJTIxJTVCZW5kaWYlNUQtLSUzRSUwQSsrKysrKyUzQyUyMS0tJTVCaWYrJTI4bXNvJTI5JTdDJTI4SUUlMjklNUQlM0UlM0MlMkZ0ciUzRSUzQyUyRnRhYmxlJTNFJTNDJTJGdGQlM0UlM0MlMkZ0ciUzRSUzQyUyRnRhYmxlJTNFJTNDJTIxJTVCZW5kaWYlNUQtLSUzRSUwQSsrKyslM0MlMkZkaXYlM0UlMEErKyUzQyUyRmRpdiUzRSUwQSsrJTNDJTJGZGl2JTNFJTBBKyslMEElMEElMEErKysrJTNDJTIxLS0lNUJpZislMjhtc28lMjklN0MlMjhJRSUyOSU1RCUzRSUzQyUyRnRkJTNFJTNDJTJGdHIlM0UlM0MlMkZ0YWJsZSUzRSUzQyUyMSU1QmVuZGlmJTVELS0lM0UlMEErKysrJTNDJTJGdGQlM0UlMEErKyUzQyUyRnRyJTNFJTBBKyslM0MlMkZ0Ym9keSUzRSUwQSsrJTNDJTJGdGFibGUlM0UlMEErKyUzQyUyMS0tJTVCaWYrbXNvJTVEJTNFJTNDJTJGZGl2JTNFJTNDJTIxJTVCZW5kaWYlNUQtLSUzRSUwQSsrJTNDJTIxLS0lNUJpZitJRSU1RCUzRSUzQyUyRmRpdiUzRSUzQyUyMSU1QmVuZGlmJTVELS0lM0UlMEElM0MlMkZib2R5JTNFJTBBJTBBJTNDJTJGaHRtbCUzRSUwQQo="
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
