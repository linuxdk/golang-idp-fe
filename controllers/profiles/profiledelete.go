package profiles

import (
  "strings"
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  "github.com/gorilla/csrf"
  "github.com/gin-contrib/sessions"
  idp "github.com/charmixer/idp/client"

  "github.com/charmixer/idpui/app"
  "github.com/charmixer/idpui/config"
  "github.com/charmixer/idpui/environment"
  "github.com/charmixer/idpui/utils"
)

type profileDeleteForm struct {
  RiskAccepted string `form:"risk_accepted"`
}

func ShowProfileDelete(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "ShowProfileDelete",
    })

    identity := app.RequireIdentity(c)
    if identity == nil {
      log.Debug("Missing Identity")
      c.AbortWithStatus(http.StatusForbidden)
      return
    }

    session := sessions.Default(c)

    riskAccepted := session.Get("profiledelete.risk_accepted")

    errors := session.Flashes("profiledelete.errors")
    err := session.Save() // Remove flashes read, and save submit fields
    if err != nil {
      log.Debug(err.Error())
    }

    var errorRiskAccepted string

    if len(errors) > 0 {
      errorsMap := errors[0].(map[string][]string)
      for k, v := range errorsMap {

        if k == "errorRiskAccepted" && len(v) > 0 {
          errorRiskAccepted = strings.Join(v, ", ")
        }

      }
    }

    c.HTML(http.StatusOK, "profiledelete.html", gin.H{
      "title": "Delete profile",
      "links": []map[string]string{
        {"href": "/public/css/dashboard.css"},
      },
      csrf.TemplateTag: csrf.TemplateField(c.Request),
      "username": identity.Subject,
      "name": identity.Name,
      "RiskAccepted": riskAccepted,
      "errorRiskAccepted": errorRiskAccepted,
      "profileDeleteUrl": "/me/delete",
    })
  }
  return gin.HandlerFunc(fn)
}

func SubmitProfileDelete(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "SubmitProfileDelete",
    })

    var form profileDeleteForm
    err := c.Bind(&form)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    identity := app.RequireIdentity(c)
    if identity == nil {
      log.Debug("Missing Identity")
      c.AbortWithStatus(http.StatusForbidden)
      return
    }

    session := sessions.Default(c)
    errors := make(map[string][]string)

    riskAccepted := len(form.RiskAccepted) > 0

    if riskAccepted == false {
      errors["errorRiskAccepted"] = append(errors["errorRiskAccepted"], "You have not accepted the risk")
    }

    if len(errors) <= 0  {
      if riskAccepted == true {

        idpClient := app.IdpClientUsingAuthorizationCode(env, c)

        deleteRequest := &idp.IdentitiesDeleteRequest{
          Id: identity.Id,
        }
        deleteChallenge, err := idp.DeleteIdentity(idpClient, config.GetString("idp.public.url") + config.GetString("idp.public.endpoints.identities"), deleteRequest)
        if err != nil {
          log.Debug(err.Error())
          c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
          c.Abort()
          return
        }

        // Cleanup session
        session.Delete("profiledelete.risk_accepted")
        session.Delete("profiledelete.errors")
        err = session.Save()
        if err != nil {
          log.Debug(err.Error())
        }

        redirectTo := deleteChallenge.RedirectTo
        log.WithFields(logrus.Fields{"redirect_to": redirectTo}).Debug("Redirecting");
        c.Redirect(http.StatusFound, redirectTo)
        c.Abort()
        return
      }
    }

    // Deny by default
    session.Set("profiledelete.risk_accepted", form.RiskAccepted)
    session.AddFlash(errors, "profiledelete.errors")
    err = session.Save()
    if err != nil {
      log.Debug(err.Error())
    }

    submitUrl, err := utils.FetchSubmitUrlFromRequest(c.Request, nil)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }
    log.WithFields(logrus.Fields{"redirect_to": submitUrl}).Debug("Redirecting")
    c.Redirect(http.StatusFound, submitUrl)
    c.Abort()
    return

  }
  return gin.HandlerFunc(fn)
}
