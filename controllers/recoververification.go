package controllers

import (
  "net/http"
  "strings"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  "github.com/gorilla/csrf"
  "github.com/gin-contrib/sessions"
  "golang-idp-fe/config"
  "golang-idp-fe/environment"
  "golang-idp-fe/gateway/idpapi"
)

type verificationForm struct {
  Challenge        string `form:"recover_challenge" binding:"required"`
  VerificationCode string `form:"verification_code" binding:"required"`
  Password         string `form:"password" binding:"required"`
  PasswordRetyped  string `form:"password_retyped" binding:"required"`
}

func ShowRecoverVerification(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "ShowRecoverVerification",
    })

    recoverChallenge := c.Query("recover_challenge")
    if recoverChallenge == "" {
      log.WithFields(logrus.Fields{
        "recover_challenge": recoverChallenge,
      }).Debug("Missing recover_challenge")
      c.JSON(http.StatusNotFound, gin.H{"error": "Missing recover_challenge"})
      c.Abort()
      return
    }

    session := sessions.Default(c)

    errors := session.Flashes("recoververification.errors")
    err := session.Save() // Remove flashes read, and save submit fields
    if err != nil {
      log.Debug(err.Error())
    }

    var errorVerificationCode string
    var errorPassword string
    var errorPasswordRetyped string

    if len(errors) > 0 {
      errorsMap := errors[0].(map[string][]string)
      for k, v := range errorsMap {

        if k == "errorVerificationCode" && len(v) > 0 {
          errorVerificationCode = strings.Join(v, ", ")
        }
        if k == "errorPassword" && len(v) > 0 {
          errorPassword = strings.Join(v, ", ")
        }
        if k == "errorPasswordRetyped" && len(v) > 0 {
          errorPasswordRetyped = strings.Join(v, ", ")
        }

      }
    }

    c.HTML(http.StatusOK, "recoververification.html", gin.H{
      csrf.TemplateTag: csrf.TemplateField(c.Request),
      "errorVerificationCode": errorVerificationCode,
      "errorPassword": errorPassword,
      "errorPasswordRetyped": errorPasswordRetyped,
      "recoverChallenge": recoverChallenge,
    })
  }
  return gin.HandlerFunc(fn)
}

func SubmitRecoverVerification(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "SubmitRecoverVerification",
    })

    var form verificationForm
    err := c.Bind(&form)
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    session := sessions.Default(c)

    errors := make(map[string][]string)

    verificationCode := strings.TrimSpace(form.VerificationCode)
    if verificationCode == "" {
      errors["errorVerificationCode"] = append(errors["errorVerificationCode"], "Missing verification code")
    }

    log.WithFields(logrus.Fields{"fixme": 1}).Debug("Should we trim password?")
    password := strings.TrimSpace(form.Password)
    if password == "" {
      errors["errorPassword"] = append(errors["errorPassword"], "Missing password")
    }

    retypedPassword := strings.TrimSpace(form.PasswordRetyped)
    if retypedPassword == "" {
      errors["errorPasswordRetyped"] = append(errors["errorPasswordRetyped"], "Missing password")
    }

    if retypedPassword != password {
      errors["errorPasswordRetyped"] = append(errors["errorPasswordRetyped"], "Must match password")
    }

    if len(errors) > 0 {
      session.AddFlash(errors, "recoververification.errors")
      err = session.Save()
      if err != nil {
        log.Debug(err.Error())
      }
      redirectTo := c.Request.URL.RequestURI()
      log.WithFields(logrus.Fields{"redirect_to": redirectTo}).Debug("Redirecting")
      c.Redirect(http.StatusFound, redirectTo)
      c.Abort();
      return
    }

    idpapiClient := idpapi.NewIdpApiClient(env.IdpApiConfig)

    recoverRequest := idpapi.RecoverVerificationRequest{
      Challenge: form.Challenge,
      VerificationCode: form.VerificationCode,
      Password: form.Password,
    }
    recoverResponse, err := idpapi.RecoverVerification(config.GetString("idpapi.public.url") + config.GetString("idpapi.public.endpoints.recoververification"), idpapiClient, recoverRequest)
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    log.WithFields(logrus.Fields{
      "redirect_to": recoverResponse.RedirectTo,
    }).Debug("Redirecting");
    c.Redirect(http.StatusFound, recoverResponse.RedirectTo)
  }
  return gin.HandlerFunc(fn)
}